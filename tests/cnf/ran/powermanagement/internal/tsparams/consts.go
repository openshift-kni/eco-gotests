package tsparams

import (
	"time"

	"github.com/golang/glog"
)

const (
	// LabelSuite is the label for all the tests in this suite.
	LabelSuite = "powermanagement"
	// LabelPowerSaveTestCases is the label for a particular test case.
	LabelPowerSaveTestCases = "powersave"
	// PowerSaveTimeout is the timeout being used in powersave tests.
	PowerSaveTimeout = 10 * time.Minute
	// TestingNamespace is the tests namespace.
	TestingNamespace = "ran-test"
	// PrivPodNamespace is the privileged pod namespace.
	PrivPodNamespace = "cnf-ran-gotests-priv"

	// PowerSavingMode is the name of the power saving power state.
	PowerSavingMode = "powersaving"
	// PerformanceMode is the name of the performance power state.
	PerformanceMode = "performance"
	// HighPerformanceMode is the name of the high performance power state.
	HighPerformanceMode = "highperformance"

	// IpmiDcmiPowerMinimumDuringSampling is the minimum power metric.
	IpmiDcmiPowerMinimumDuringSampling = "minPower"
	// IpmiDcmiPowerMaximumDuringSampling is the maximum power metric.
	IpmiDcmiPowerMaximumDuringSampling = "maxPower"
	// IpmiDcmiPowerAverageDuringSampling is the average power metric.
	IpmiDcmiPowerAverageDuringSampling = "avgPower"
	// IpmiDcmiPowerInstantaneous is the instantaneous power metric.
	IpmiDcmiPowerInstantaneous = "instantaneousPower"

	// RanPowerMetricTotalSamples is the metric for total samples.
	RanPowerMetricTotalSamples = "ranmetrics_power_total_samples"
	// RanPowerMetricSamplingIntervalSeconds is the metric for sampling interval.
	RanPowerMetricSamplingIntervalSeconds = "ranmetrics_power_sampling_interval_seconds"
	// RanPowerMetricMinInstantPower is the metric for minimum instantaneous power.
	RanPowerMetricMinInstantPower = "ranmetrics_power_min_instantaneous"
	// RanPowerMetricMaxInstantPower is the metric for maximum instantaneous power.
	RanPowerMetricMaxInstantPower = "ranmetrics_power_max_instantaneous"
	// RanPowerMetricMeanInstantPower is the metric for mean instantaneous power.
	RanPowerMetricMeanInstantPower = "ranmetrics_power_mean_instantaneous"
	// RanPowerMetricStdDevInstantPower is the metric for standard deviation of instantaneous power.
	RanPowerMetricStdDevInstantPower = "ranmetrics_power_standard_deviation_instantaneous"
	// RanPowerMetricMedianInstantPower is the metric for median instantaneous power.
	RanPowerMetricMedianInstantPower = "ranmetrics_power_median_instantaneous"

	// LogLevel is the verbosity of glog statements in this test suite.
	LogLevel glog.Level = 90
)
