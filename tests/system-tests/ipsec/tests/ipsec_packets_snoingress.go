package ipsec_system_test

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/system-tests/internal/sshcommand"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/system-tests/ipsec/internal/iperf3workload"
	. "github.com/rh-ecosystem-edge/eco-gotests/tests/system-tests/ipsec/internal/ipsecinittools"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/system-tests/ipsec/internal/ipsecparams"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/system-tests/ipsec/internal/ipsectunnel"
)

// The iperf3 client and server should be started as follows for this test case:
//
// SNO cluster pod (iperf3 server):  iperf3 -s -p 30000 -J
//                                               ^^^^^^^
//                                           Service NodePort
//
// SecurityGateway  (iperf3 client): iperf3 -c 10.1.232.10 -B 172.16.123.10 -p 30000  -n 500M
//                                             ^^^^^^^^^^^    ^^^^^^^^^^^^^   ^^^^^^^
//                                               SNO IP       SecGW VPN IP   Service NodePort
//

var _ = Describe(
	"IpsecPacketsSnoIngress",
	Label("IpsecPacketsSnoIngress"),
	Ordered,
	ContinueOnFailure,
	func() {
		BeforeAll(func() {
			nodePort, err := strconv.Atoi(IpsecTestConfig.NodePort)
			Expect(err).ToNot(HaveOccurred(), "Error convering IpsecTestConfig.NodePort")

			_, err = iperf3workload.CreateService(APIClient, int32(nodePort))
			Expect(err).ToNot(HaveOccurred(), "Error while creating iperf3 service")

			nodeNames, _ := GetNodeNames()
			_, err = iperf3workload.CreateWorkload(APIClient,
				nodeNames[0],
				IpsecTestConfig.Iperf3ToolImage)
			Expect(err).ToNot(HaveOccurred(), "Error while deploying iperf3 workload")
		})

		It("Asserts packets received on the SNO cluster are sent via IPSec", func() {
			glog.V(ipsecparams.IpsecLogLevel).Infof("Check packets received on the SNO cluster are sent via IPSec")

			nodeNames, _ := GetNodeNames()

			// Channels for asynchronous calls below
			iperf3ServerChannel := make(chan bool)
			sshChannel := make(chan *sshcommand.SSHCommandResult)

			// Asynchronously start the iperf3 server on the SNO pod
			go func(channel chan bool) {
				// Start the iperf3 app in server mode
				iperf3ServerCmd := append(slices.Clone(ipsecparams.Iperf3ServerBaseCmd),
					ipsecparams.Iperf3OptionPort,
					IpsecTestConfig.NodePort)
				channel <- iperf3workload.LaunchIperf3Command(APIClient, iperf3ServerCmd)
			}(iperf3ServerChannel)

			// Sleep to let the iperf3 server get started before trying to start
			// the iperf3 client
			time.Sleep(10 * time.Second)

			// Verify the number of packets received
			packetsBefore := ipsectunnel.TunnelPackets(nodeNames[0])

			// Start the iperf3 client Asynchronously
			go func(channel chan *sshcommand.SSHCommandResult) {
				// The iperf3 server-mode command
				iperf3ClientCmd := append(slices.Clone(ipsecparams.Iperf3ClientBaseCmd),
					IpsecTestConfig.Iperf3ServerSnoIP,
					ipsecparams.Iperf3OptionBind,
					IpsecTestConfig.SecGwServerIP,
					ipsecparams.Iperf3OptionPort,
					IpsecTestConfig.NodePort,
					ipsecparams.Iperf3OptionBytes,
					IpsecTestConfig.Iperf3ClientTxBytes)
				sshAddrStr := fmt.Sprintf("%s:%s",
					IpsecTestConfig.SecGwHostIP,
					IpsecTestConfig.SSHPort)
				// For this ssh to work, the ssh privateKey has to have already been
				// added to the SecGW server ~/.ssh/authorized_keys file
				sshResult := sshcommand.SSHCommand(strings.Join(iperf3ClientCmd, " "),
					sshAddrStr,
					IpsecTestConfig.SSHUser,
					IpsecTestConfig.SSHPrivateKey)
				channel <- sshResult
			}(sshChannel)

			// Get the client via the channel
			clientOutput := <-sshChannel
			Expect(clientOutput.Err == nil).To(BeTrue(), "Error in iperf3 client execution: %v, %v",
				clientOutput.Err, clientOutput.SSHOutput)
			packetsAfter := ipsectunnel.TunnelPackets(nodeNames[0])

			// Get the server results via the channel
			serverOutput := <-iperf3ServerChannel
			Expect(serverOutput).To(BeTrue(), "Error in iperf3 server execution.")

			Expect(packetsBefore != nil).To(BeTrue(), "Error unable to get packetsBefore.")
			Expect(packetsAfter != nil).To(BeTrue(), "Error unable to get packetsAfter.")

			glog.V(ipsecparams.IpsecLogLevel).Infof("Packets Before in/outBytes %d/%d, After in/outBytes %d/%d",
				packetsBefore.InBytes, packetsBefore.OutBytes,
				packetsAfter.InBytes, packetsAfter.OutBytes)

			// Need to verify the packet counts better
			Expect(packetsAfter.InBytes-packetsBefore.InBytes).To(BeNumerically(">", 0),
				"Invalid number of InBytes")
		})

		AfterAll(func() {
			// Stop the workload
			err := iperf3workload.DeleteWorkload(APIClient)
			Expect(err).ToNot(HaveOccurred(), "Error in DeleteWorkload")

			err = iperf3workload.DeleteService(APIClient)
			Expect(err).ToNot(HaveOccurred(), "Error in DeleteService")
		})
	},
)
