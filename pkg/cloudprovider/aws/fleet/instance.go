/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package fleet

import (
	"context"
	"fmt"
	"math/rand"
	"strings"

	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
)

type InstanceProvider struct {
	ec2 ec2iface.EC2API
	vpc *VPCProvider
}

// Create an instance given the constraints.
func (p *InstanceProvider) Create(ctx context.Context,
	launchTemplate *ec2.LaunchTemplate,
	instanceTypeOptions []string,
	zonalSubnetOptions map[string][]*ec2.Subnet,
) (*string, error) {
	// 1. Construct override options.
	var overrides []*ec2.FleetLaunchTemplateOverridesRequest
	for _, instanceType := range instanceTypeOptions {
		for zone, subnets := range zonalSubnetOptions {
			overrides = append(overrides, &ec2.FleetLaunchTemplateOverridesRequest{
				AvailabilityZone: aws.String(zone),
				InstanceType:     aws.String(instanceType),
				// FleetAPI cannot span subnets from the same AZ, so randomize.
				SubnetId: aws.String(*subnets[rand.Intn(len(subnets))].SubnetId),
			})
		}
	}

	// 2. Create fleet
	createFleetOutput, err := p.ec2.CreateFleetWithContext(ctx, &ec2.CreateFleetInput{
		Type: aws.String(ec2.FleetTypeInstant),
		TargetCapacitySpecification: &ec2.TargetCapacitySpecificationRequest{
			DefaultTargetCapacityType: aws.String(ec2.DefaultTargetCapacityTypeOnDemand),
			TotalTargetCapacity:       aws.Int64(1),
		},
		LaunchTemplateConfigs: []*ec2.FleetLaunchTemplateConfigRequest{{
			LaunchTemplateSpecification: &ec2.FleetLaunchTemplateSpecificationRequest{
				LaunchTemplateName: launchTemplate.LaunchTemplateName,
				Version:            aws.String("$Default"),
			},
			Overrides: overrides,
		}},
	})
	if err != nil {
		return nil, fmt.Errorf("creating fleet %w", err)
	}
	// TODO aggregate errors
	if count := len(createFleetOutput.Errors); count > 0 {
		return nil, fmt.Errorf("errors while creating fleet, %v", createFleetOutput.Errors)
	}
	if count := len(createFleetOutput.Instances); count != 1 {
		return nil, fmt.Errorf("expected 1 instance, but got %d", count)
	}
	if count := len(createFleetOutput.Instances[0].InstanceIds); count != 1 {
		return nil, fmt.Errorf("expected 1 instance ids, but got %d", count)
	}
	return createFleetOutput.Instances[0].InstanceIds[0], nil
}

func (p *InstanceProvider) Terminate(ctx context.Context, nodes []*v1.Node) error {
	ids := p.getInstanceIDs(nodes)

	_, err := p.ec2.TerminateInstancesWithContext(ctx, &ec2.TerminateInstancesInput{
		InstanceIds: ids,
	})
	if err != nil {
		return fmt.Errorf("terminating %d instances, %w", len(ids), err)
	}

	return nil
}

func (p *InstanceProvider) getInstanceIDs(nodes []*v1.Node) []*string {
	ids := []*string{}
	for _, node := range nodes {
		id := strings.Split(node.Spec.ProviderID, "/")
		if len(id) < 5 {
			zap.S().Debugf("Continuing after failure to parse instance id, %s has invalid format", node.Name)
			continue
		}
		ids = append(ids, aws.String(id[4]))
	}
	return ids
}
