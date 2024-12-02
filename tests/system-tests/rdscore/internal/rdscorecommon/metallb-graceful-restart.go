package rdscorecommon

import (
	"fmt"
	"net"
	"net/netip"
	"strconv"
	"sync"
	"time"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-goinfra/pkg/service"
	. "github.com/openshift-kni/eco-gotests/tests/system-tests/rdscore/internal/rdscoreinittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/rdscore/internal/rdscoreparams"
)

// Event structure stores basic data about connection attempts.
type Event struct {
	Msg       string
	Timestamp time.Time
}

// Counter structre is used to measure successful and failed connection attempts.
type Counter struct {
	Count       int64
	CounterLock sync.Mutex
	Start       time.Time
	End         time.Time
	Success     []Event
	Failures    []Event
	FailedDial  []Event
	FailedRead  []Event
	FailedWrite []Event
}

type myData struct {
	TCPConn *net.TCPConn
	TCPErr  error
}

// mTCPConnect dials to the TCP endpoint and terminates within a specied timeout.
func mTCPConnect(addr *net.TCPAddr, timeOut int) (*net.TCPConn, error) {
	result := make(chan myData)

	go func(network string, laddr, raddr *net.TCPAddr, result chan myData) {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Trying to connect to %q at %v", raddr.String(),
			time.Now().UnixMilli())

		tConn, err := net.DialTCP(network, laddr, addr)

		if err != nil {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Failed to connect to %q due to: %v",
				raddr.String(), err)

			result <- myData{TCPConn: nil, TCPErr: err}
		}

		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Connected to %q", raddr.String())

		result <- myData{TCPConn: tConn, TCPErr: nil}
	}("tcp", nil, addr, result)

	select {
	case data := <-result:
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Read from connection: %v\n", data)

		return data.TCPConn, data.TCPErr
	case <-time.After(time.Duration(timeOut) * time.Millisecond):
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("timeout waiting for connection to be establised")

		return nil, fmt.Errorf("timeout waiting for connection to be establised")
	}
}

// verifySingleTCPConnection validates graceful restart feature by opening a TCP connection
// to the IP address of LoadBalancer service and using the connection to
// send/receive data. During the run frr-k8s pod is destroyed but this
// shouldn't result in the connection being dropped.
//
//nolint:funlen
func verifySingleTCPConnection(loadBalancerIP string, servicePort int32,
	finished, ready chan bool, stats *Counter) {
	var (
		err      error
		endPoint string
	)

	By(fmt.Sprintf("Accessing backend pods via %s IP", loadBalancerIP))

	myIP, err := netip.ParseAddr(loadBalancerIP)

	Expect(err).ToNot(HaveOccurred(), "Failed to parse IP address")

	if myIP.Is4() {
		endPoint = fmt.Sprintf("%s:%d", loadBalancerIP, servicePort)
	}

	if myIP.Is6() {
		endPoint = fmt.Sprintf("[%s]:%d", loadBalancerIP, servicePort)
	}

	getHTTPMsg := []byte(fmt.Sprintf("GET / HTTP/1.1\r\nHost: %s\r\n\r\n", endPoint))

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Resolving TCP endpoint %q", endPoint)

	addr, err := net.ResolveTCPAddr("tcp", endPoint)

	if err != nil {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Failed resolve TCP address %q : %v", endPoint, err)

		ready <- true

		return
	}

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Dialing to the TCP endpoint %q", endPoint)

	lbConnection, err := mTCPConnect(addr, int(1000))

	if err != nil {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Failed dailing to %q : %v", endPoint, err)

		ready <- true

		return
	}

	defer lbConnection.Close()

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Enabling KeepAlive for TCP connection")

	err = lbConnection.SetKeepAlive(true)

	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("failed to enable KeepAlive for the connection: %v", err))

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Setting KeepAlivePeriod to 1s")

	err = lbConnection.SetKeepAlivePeriod(time.Duration(1))

	Expect(err).ToNot(HaveOccurred(),
		fmt.Sprintf("failed to enable KeepAlivePeriod for the connection: %v", err))

	var stop bool

	for !stop {
		select {
		case <-finished:
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Received data on 'finishd' channel")

			stop = true
		default:
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Writing to the TCP connection at %v", time.Now().UnixMilli())

			nSent, err := lbConnection.Write(getHTTPMsg)

			if err != nil || nSent == 0 {
				glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Failed to write to connection: %v", err)

				stats.CounterLock.Lock()
				stats.Count++
				stats.Failures = append(stats.Failures,
					Event{Msg: fmt.Sprintf("Failed to write to connection: %v", err), Timestamp: time.Now()})
				stats.FailedWrite = append(stats.FailedWrite,
					Event{Msg: fmt.Sprintf("Failed to write to connection: %v", err), Timestamp: time.Now()})
				stats.CounterLock.Unlock()

				stop = true

				continue
			}

			err = lbConnection.SetReadDeadline(time.Now().Add(300 * time.Millisecond))

			if err != nil {
				glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Failed set ReadDeadline: %v", err)

				stats.CounterLock.Lock()
				stats.Count++
				stats.Failures = append(stats.Failures,
					Event{Msg: fmt.Sprintf("Failed to set ReadDeadline %v", err), Timestamp: time.Now()})
				stats.CounterLock.Unlock()

				stop = true

				continue
			}

			msgReply := make([]byte, 1024)

			bRead, err := lbConnection.Read(msgReply)

			if err != nil {
				glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Failed to read reply(%v): %v",
					time.Now().UnixMilli(), err)
				stats.CounterLock.Lock()
				stats.Count++
				stats.Failures = append(stats.Failures,
					Event{Msg: fmt.Sprintf("Failed to read reply: %v", err), Timestamp: time.Now()})
				stats.FailedRead = append(stats.FailedRead,
					Event{Msg: fmt.Sprintf("Failed to read reply: %v", err), Timestamp: time.Now()})
				stats.CounterLock.Unlock()

				stop = true

				continue
			} else {
				glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Read %d bytes", bRead)

				stats.CounterLock.Lock()
				stats.Count++
				stats.Success = append(stats.Success,
					Event{Msg: string(msgReply[0:bRead]), Timestamp: time.Now()})
				stats.CounterLock.Unlock()
			}
		}
	}

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("go routine 'verifySingleTCPConnection' finished")

	ready <- true
}

// verifyMultipleTCPConnections simulates new clients connecting to the specified endpoint,
// while Metallb-FRR pod is killed.
// It opens new TCP connection to the specified endpoint during every iteration.
// 'finished' channel is used receive signal that metallb-frr pod was restarted
// 'ready' channel is used to communicate to the parent function that it's finished.
//
//nolint:funlen
func verifyMultipleTCPConnections(loadBalancerIP string, servicePort int32,
	finished, ready chan bool, stats *Counter) {
	var (
		err      error
		endPoint string
	)

	By(fmt.Sprintf("Accessing backend pods via %s IP", loadBalancerIP))

	myIP, err := netip.ParseAddr(loadBalancerIP)

	Expect(err).ToNot(HaveOccurred(), "Failed to parse IP address")

	if myIP.Is4() {
		endPoint = fmt.Sprintf("%s:%d", loadBalancerIP, servicePort)
	}

	if myIP.Is6() {
		endPoint = fmt.Sprintf("[%s]:%d", loadBalancerIP, servicePort)
	}

	getHTTPMsg := []byte(fmt.Sprintf("GET / HTTP/1.1\r\nHost: %s\r\n\r\n", endPoint))

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Resolving TCP endpoint %q", endPoint)

	addr, err := net.ResolveTCPAddr("tcp", endPoint)

	if err != nil {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Failed resolve TCP address %q : %v", endPoint, err)

		ready <- true

		return
	}

	var stop bool

	for !stop {
		select {
		case <-finished:
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Received data on 'finished' channel")

			stop = true
		default:
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Dialing to the TCP endpoint %q", endPoint)

			lbConnection, err := mTCPConnect(addr, int(300))

			if err != nil {
				glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Failed dailing to %q : %v", endPoint, err)

				stats.CounterLock.Lock()
				stats.Count++
				stats.Failures = append(stats.Failures,
					Event{Msg: fmt.Sprintf("Failed dailing to %s : %v", endPoint, err), Timestamp: time.Now()})
				stats.FailedDial = append(stats.FailedDial,
					Event{Msg: fmt.Sprintf("Failed dailing to %s : %v", endPoint, err), Timestamp: time.Now()})
				stats.CounterLock.Unlock()

				continue
			}

			addrMsg := fmt.Sprintf("-> Successfully connected to %s from %q",
				lbConnection.RemoteAddr(), lbConnection.LocalAddr())

			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("\t%s", addrMsg)

			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Writing to the TCP connection at %v", time.Now().UnixMilli())

			nSent, err := lbConnection.Write(getHTTPMsg)

			if err != nil || nSent == 0 {
				glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Failed to write to connection: %v", err)

				stats.CounterLock.Lock()
				stats.Count++
				stats.Failures = append(stats.Failures,
					Event{Msg: fmt.Sprintf("Failed to write to connection: %v", err), Timestamp: time.Now()})
				stats.FailedWrite = append(stats.FailedWrite,
					Event{Msg: fmt.Sprintf("Failed to write to connection: %v", err), Timestamp: time.Now()})
				stats.CounterLock.Unlock()

				lbConnection.Close()

				continue
			}

			err = lbConnection.SetReadDeadline(time.Now().Add(300 * time.Millisecond))

			if err != nil {
				glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Failed set ReadDeadline: %v", err)

				stats.CounterLock.Lock()
				stats.Count++
				stats.Failures = append(stats.Failures,
					Event{Msg: fmt.Sprintf("Failed to set ReadDeadline %v", err), Timestamp: time.Now()})
				stats.CounterLock.Unlock()

				lbConnection.Close()

				continue
			}

			msgReply := make([]byte, 1024)

			bRead, err := lbConnection.Read(msgReply)

			if err != nil {
				glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Failed to read reply(%v): %v",
					time.Now().UnixMilli(), err)
				stats.CounterLock.Lock()
				stats.Count++
				stats.Failures = append(stats.Failures,
					Event{Msg: fmt.Sprintf("Failed to read reply: %v", err), Timestamp: time.Now()})
				stats.FailedRead = append(stats.FailedRead,
					Event{Msg: fmt.Sprintf("Failed to read reply: %v", err), Timestamp: time.Now()})
				stats.CounterLock.Unlock()

				lbConnection.Close()

				continue
			} else {
				glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Read %d bytes from TCP connection", bRead)

				stats.CounterLock.Lock()
				stats.Count++
				stats.Success = append(stats.Success,
					Event{Msg: string(msgReply[0:bRead]), Timestamp: time.Now()})
				stats.CounterLock.Unlock()
			}

			lbConnection.Close()
		}
	}

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("go routine 'verifyMultipleTCPConnections' finished")

	stats.CounterLock.Lock()
	stats.End = time.Now()
	stats.CounterLock.Unlock()

	ready <- true
}

// restartMetallbFRRPod finds metallb-frr pod running on a node,
// deletes it, and waits for a new pod to start and reach ready state.
//
//nolint:funlen
func restartMetallbFRRPod(node string, metallbFRRRestartFailed *bool, finished chan bool) {
	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Looking for a metallb-frr pod on %q node", node)

	mPodList, err := pod.List(APIClient, rdscoreparams.MetalLBOperatorNamespace,
		metav1.ListOptions{LabelSelector: rdscoreparams.MetalLBFRRPodSelector})

	if len(mPodList) == 0 {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Failed to found pod: %v", err)

		*metallbFRRRestartFailed = true

		finished <- true

		return
	}

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Filter pod running on %q", node)

	var prevPod *pod.Builder

	for _, _pod := range mPodList {
		if _pod.Definition.Spec.NodeName == node {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Found pod %q running on %q",
				_pod.Definition.Name, node)

			prevPod = _pod

			break
		}
	}

	if prevPod == nil {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Failed to found frr-k8s pod running on %q", node)

		*metallbFRRRestartFailed = true

		finished <- true

		return
	}

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Deleting pod %q", prevPod.Definition.Name)

	prevPod, err = prevPod.DeleteAndWait(15 * time.Second)

	if err != nil {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Failed to delete pod %q: %v",
			prevPod.Definition.Name, err)

		*metallbFRRRestartFailed = true

		finished <- true

		return
	}

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Looking for a new metallb-frr pod")

	var newPod *pod.Builder

	for i := 0; i < 60; i++ {
		mPodList, err = pod.List(APIClient, "metallb-system",
			metav1.ListOptions{LabelSelector: "app=frr-k8s"})

		if len(mPodList) == 0 {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Failed to found pod: %v", err)

			*metallbFRRRestartFailed = true

			time.Sleep(1 * time.Second)

			continue
		}

		for _, _pod := range mPodList {
			if _pod.Definition.Spec.NodeName == node {
				glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Found pod running on %q", node)

				newPod = _pod

				break
			}
		}

		if newPod.Definition.Name == prevPod.Definition.Name {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("No new frr-k8s pod found")

			*metallbFRRRestartFailed = true
			newPod = nil

			time.Sleep(1 * time.Second)

			continue
		} else {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("\t\t---> New frr-k8s pod found: %q",
				newPod.Definition.Name)

			*metallbFRRRestartFailed = false

			break
		}
	}

	if newPod == nil {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("No new frr-k8s pod found on %q node", node)

		*metallbFRRRestartFailed = true

		finished <- true

		return
	}

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Waiting for pod %q to be Ready",
		newPod.Definition.Name)

	err = newPod.WaitUntilReady(3 * time.Minute)

	if err != nil {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Pod %q hasn't reached Ready state",
			newPod.Definition.Name)

		*metallbFRRRestartFailed = true
	}

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Goroutine restartMetallbFRRPod finished")

	finished <- true
}

//nolint:funlen,gocognit
func verifyGracefulRestartFlow(svcName string, checkIPv6 bool, checkMultipleConnections bool) {
	By(fmt.Sprintf("Pulling %q service configuration", svcName))

	var (
		svcBuilder *service.Builder
		err        error
		ctx        SpecContext
	)

	Eventually(func() bool {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Pulling %q service from %q namespace",
			svcName, RDSCoreConfig.GracefulRestartServiceNS)

		svcBuilder, err = service.Pull(APIClient, svcName, RDSCoreConfig.GracefulRestartServiceNS)

		if err != nil {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Error pulling %q service from %q namespace: %v",
				svcName, RDSCoreConfig.GracefulRestartServiceNS, err)

			return false
		}

		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Successfully pulled %q service from %q namespace",
			svcBuilder.Definition.Name, svcBuilder.Definition.Namespace)

		return true
	}).WithContext(ctx).WithPolling(5*time.Second).WithTimeout(1*time.Minute).Should(BeTrue(),
		fmt.Sprintf("Error obtaining service %q configuration", svcName))

	By(fmt.Sprintf("Asserting service %q has LoadBalancer IP address", svcBuilder.Definition.Name))

	Eventually(func() bool {
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Check service %q in %q namespace has LoadBalancer IP",
			svcBuilder.Definition.Name, svcBuilder.Definition.Namespace)

		refreshSVC := svcBuilder.Exists()

		if !refreshSVC {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Failed to refresh service status")

			return false
		}

		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Service has %d IP addresses",
			len(svcBuilder.Object.Status.LoadBalancer.Ingress))

		return len(svcBuilder.Object.Status.LoadBalancer.Ingress) != 0
	}).WithContext(ctx).WithPolling(15*time.Second).WithTimeout(3*time.Minute).Should(BeTrue(),
		"Service does not have LoadBalancer IP address")

	mPodList, err := pod.List(APIClient, RDSCoreConfig.GracefulRestartServiceNS,
		metav1.ListOptions{LabelSelector: RDSCoreConfig.GracefulRestartAppLabel})

	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to find pods due to: %v", err))

	Expect(len(mPodList)).ToNot(BeZero(), "0 pods found")
	Expect(len(mPodList)).To(Equal(1), "More then 1 pod found")

	nodeName := mPodList[0].Object.Spec.NodeName

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Will crash metallb-frr pod on %q node during a test", nodeName)

	finished := make(chan bool)
	ready := make(chan bool)

	statistics := Counter{Start: time.Now()}

	for _, vip := range svcBuilder.Object.Status.LoadBalancer.Ingress {
		loadBalancerIP := vip.IP

		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Accessing  workload via LoadBalancer's IP %s", loadBalancerIP)

		myIP, err := netip.ParseAddr(loadBalancerIP)

		Expect(err).ToNot(HaveOccurred(), "Failed to parse IP address")

		if myIP.Is6() && !checkIPv6 {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Skipping IPv6 addrress")

			continue
		}

		if myIP.Is4() && checkIPv6 {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Skipping IPv4 address")

			continue
		}

		normalizedPort, err := strconv.Atoi(RDSCoreConfig.GracefulRestartAppServicePort)

		Expect(err).ToNot(HaveOccurred(), "Failed to part service port")

		if !checkMultipleConnections {
			go verifySingleTCPConnection(loadBalancerIP, int32(normalizedPort), finished, ready, &statistics)
		} else {
			go verifyMultipleTCPConnections(loadBalancerIP, int32(normalizedPort), finished, ready, &statistics)
		}
	}

	//
	// Remove metallb-frr pod
	//

	var metallbFRRRestartFailed = false

	go restartMetallbFRRPod(nodeName, &metallbFRRRestartFailed, finished)

	// Wait for metallb-frr pod to be restarted.

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("\t\t *** Waiting for go routines to finish ***")

	select {
	case <-ready:
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("*** Received signal on 'ready' channel from go routines ***")
		close(ready)

		if metallbFRRRestartFailed {
			Fail("Error during metallb-frr pod restart")
		}

		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("\t>>> There were %d requests", statistics.Count)
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("\t>>> Failed requests: %d", len(statistics.Failures))
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("\t>>> Successful requests: %d", len(statistics.Success))

		if len(statistics.Failures) != 0 {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("\t\t>>> Failed request 0: %s", statistics.Failures[0].Msg)
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("\t\t>>> Failed request 0: %v", statistics.Failures[0].Timestamp)
			lIdx := len(statistics.Failures) - 1
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("\t\t>>> Failed request last: %s", statistics.Failures[lIdx].Msg)
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("\t\t>>> Failed request last: %v", statistics.Failures[lIdx].Timestamp)
		}

		if len(statistics.Success) != 0 {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("\t\t>>> Success request 0: %s", statistics.Success[0].Msg)
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("\t\t>>> Success request 0: %v", statistics.Success[0].Timestamp)
			lIdx := len(statistics.Success) - 1
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("\t\t>>> Success request last: %s", statistics.Success[lIdx].Msg)
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("\t\t>>> Success request last: %v", statistics.Success[lIdx].Timestamp)
		}

		if len(statistics.FailedDial) != 0 {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("\t\t>>> There were %d failed dial attempts", len(statistics.FailedDial))
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("\t\t>>> 1st failed dial: %v", statistics.FailedDial[0].Msg)
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("\t\t>>> 1st failed dial: %v", statistics.FailedDial[0].Timestamp)
			lIdx := len(statistics.FailedDial) - 1
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("\t\t>>> Last failed dial: %v", statistics.FailedDial[lIdx].Msg)
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("\t\t>>> Last failed dial: %v", statistics.FailedDial[lIdx].Timestamp)
		} else {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("\t\t>>> There were 0 failed dial attempts")
		}

		if len(statistics.FailedRead) != 0 {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("\t\t>>> There were %d failed read attempts", len(statistics.FailedRead))
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("\t\t>>> 1st failed read: %v", statistics.FailedRead[0].Msg)
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("\t\t>>> 1st failed read: %v", statistics.FailedRead[0].Timestamp)
			lIdx := len(statistics.FailedRead) - 1
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("\t\t>>> Last failed read: %v", statistics.FailedRead[lIdx].Msg)
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("\t\t>>> Last failed read: %v", statistics.FailedRead[lIdx].Timestamp)
		} else {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("\t\t>>> There were 0 failed read attempts")
		}

		if len(statistics.FailedWrite) != 0 {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("\t\t>>> There were %d failed write attempts",
				len(statistics.FailedWrite))
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("\t\t>>> 1st failed write: %v", statistics.FailedWrite[0].Msg)
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("\t\t>>> 1st failed write: %v", statistics.FailedWrite[0].Timestamp)
			lIdx := len(statistics.FailedWrite) - 1
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("\t\t>>> Last failed write: %v", statistics.FailedWrite[lIdx].Msg)
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("\t\t>>> Last failed write: %v", statistics.FailedWrite[lIdx].Timestamp)
		} else {
			glog.V(rdscoreparams.RDSCoreLogLevel).Infof("\t\t>>> There were 0 failed write attempts")
		}

		if !checkMultipleConnections {
			if len(statistics.Failures) != 0 {
				Fail(fmt.Sprintf("Failure: there was %d connection failures with single connection", len(statistics.Failures)))
			} else {
				glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Success: there was %d connection failures with single connection",
					len(statistics.Failures))
			}
		} else {
			failPercentage := (float64(len(statistics.Failures)) / float64(statistics.Count)) * 100

			if failPercentage > float64(0.500) {
				Fail(fmt.Sprintf("Failure: there were %d(%f%%) failures with multiple connections",
					len(statistics.Failures), failPercentage))
			} else {
				glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Success: there were %d(%f%%) failures with multiple connections",
					len(statistics.Failures), failPercentage)
			}
		}

	case <-time.After(15 * time.Minute):
		glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Go routines canceled after 15 minutes")

		Fail("Error checking status of MetalLB Graceful Restart due to timeout")
	}

	glog.V(rdscoreparams.RDSCoreLogLevel).Infof("Finished waiting for go routines")
}

// VerifyGRSingleConnectionIPv4ETPLocal check MetalLB graceful restart.
func VerifyGRSingleConnectionIPv4ETPLocal() {
	verifyGracefulRestartFlow(RDSCoreConfig.GracefulRestartServiceName, false, false)
}

// VerifyGRMultipleConnectionIPv4ETPLocal check MetalLB graceful restart.
func VerifyGRMultipleConnectionIPv4ETPLocal() {
	verifyGracefulRestartFlow(RDSCoreConfig.GracefulRestartServiceName, false, true)
}

// VerifyGRSingleConnectionIPv6ETPLocal check MetalLB graceful restart.
func VerifyGRSingleConnectionIPv6ETPLocal() {
	verifyGracefulRestartFlow(RDSCoreConfig.GracefulRestartServiceName, true, false)
}

// VerifyGRMultipleConnectionIPv6ETPLocal check MetalLB graceful restart.
func VerifyGRMultipleConnectionIPv6ETPLocal() {
	verifyGracefulRestartFlow(RDSCoreConfig.GracefulRestartServiceName, true, true)
}
