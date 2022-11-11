package main

import (
	"context"
	"log"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appservice/armappservice/v2"
)

type AppServicePlan struct {
	Metrics   MetricResult
	Name      string
	Plan      string
	Instances int32
	Type      string
	Uri       string
}

type ByMem []AppServicePlan

func (a ByMem) Len() int           { return len(a) }
func (a ByMem) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByMem) Less(i, j int) bool { return a[i].Metrics.memoryAverage < a[j].Metrics.memoryAverage }

// Generated from example definition: https://github.com/Azure/azure-rest-api-specs/tree/main/specification/web/resource-manager/Microsoft.Web/stable/2022-03-01/examples/ListAppServicePlans.json
func ListAppServicePlans(ctx context.Context, cred *azidentity.DefaultAzureCredential, subID string) []AppServicePlan {

	var result = []AppServicePlan{}

	client, err := armappservice.NewPlansClient(subID, cred, nil)
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}
	pager := client.NewListPager(&armappservice.PlansClientListOptions{Detailed: nil})
	for pager.More() {
		nextResult, err := pager.NextPage(ctx)
		if err != nil {
			log.Fatalf("failed to advance page: %v", err)
		}

		for _, v := range nextResult.Value {
			if *v.Kind != "functionapp" {
				var plan AppServicePlan
				plan.Name = *v.Name
				plan.Plan = *v.SKU.Name
				plan.Instances = *v.SKU.Capacity
				plan.Type = *v.Kind
				plan.Uri = trimLeftChar(*v.ID)
				result = append(result, plan)
			}
		}
	}
	return result
}
