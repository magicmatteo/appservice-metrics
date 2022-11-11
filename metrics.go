package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/monitor/armmonitor"
	"github.com/rodaine/table"
)

type MetricResult struct {
	CPUAverage    uint8
	memoryAverage uint8
}

func (m MetricResult) String() string {
	return fmt.Sprintf("CPU Average (%%): \t%d\nMemory Average (%%): \t%d", m.CPUAverage, m.memoryAverage)
}

// Generated from example definition: https://github.com/Azure/azure-rest-api-specs/tree/main/specification/monitor/resource-manager/Microsoft.Insights/stable/2018-01-01/examples/GetMetric.json
func GetMetrics(ctx context.Context, cred *azidentity.DefaultAzureCredential, uri string) MetricResult {

	client, err := armmonitor.NewMetricsClient(cred, nil)
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}

	tnow := time.Now()
	tthen := tnow.AddDate(0, 0, -7)
	res, err := client.List(ctx, uri, &armmonitor.MetricsClientListOptions{
		Timespan:    to.Ptr(fmt.Sprintf("%s/%s", tthen.UTC().Format(time.RFC3339), tnow.UTC().Format(time.RFC3339))),
		Interval:    to.Ptr("P1D"),
		Metricnames: to.Ptr("MemoryPercentage,CpuPercentage"),
		Aggregation: to.Ptr("Average,count"),
		Orderby:     to.Ptr("Average asc"),
		ResultType:  nil,
	})
	if err != nil {
		log.Fatalf("failed to finish the request: %v", err)
	}

	var result MetricResult

	for _, metric := range res.Value {
		var average float64 = 0
		for _, n := range metric.Timeseries[0].Data {
			average += *n.Average
		}

		switch *metric.Name.Value {
		case "MemoryPercentage":
			result.memoryAverage = uint8(average / float64(len(metric.Timeseries[0].Data)))
		case "CpuPercentage":
			result.CPUAverage = uint8(average / float64(len(metric.Timeseries[0].Data)))
		}
	}
	return result
}

func main() {
	if len(os.Args) < 2 {
		log.Fatalln("Error - missing parameter subscription ID - eg: metrics \"43c71976-32e8-403f-bf1d-885b0e3598b6\"")
	}
	var subID string = os.Args[1]
	ctx := context.Background()

	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Fatalf("failed to obtain a credential: %v", err)
	}

	plans := ListAppServicePlans(ctx, cred, subID)

	fmt.Printf("\nNumber of received plans: %d\n", len(plans))
	fmt.Print("Querying plans... This may take a few seconds...\n\n")

	t := time.Now()
	var wg sync.WaitGroup
	for i := range plans {
		wg.Add(1)
		i := i
		go func(p *AppServicePlan) {
			defer wg.Done()
			p.Metrics = GetMetrics(ctx, cred, p.Uri)
		}(&plans[i])
	}
	wg.Wait()
	sort.Sort(ByMem(plans))
	tbl := table.New("Name", "CPU", "MEM", "Plan", "Instances", "Type")
	for _, plan := range plans {
		tbl.AddRow(
			plan.Name,
			plan.Metrics.CPUAverage,
			plan.Metrics.memoryAverage,
			plan.Plan,
			plan.Instances,
			plan.Type)
	}
	fmt.Printf("Request took: %d ms\n\n", time.Since(t).Milliseconds())
	tbl.Print()
	fmt.Println()

}
