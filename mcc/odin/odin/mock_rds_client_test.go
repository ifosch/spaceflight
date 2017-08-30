package odin_test

import (
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/rds/rdsiface"
)

type mockRDSClient struct {
	rdsiface.RDSAPI
	dbInstancesEndpoints map[string]rds.Endpoint
	dbInstanceSnapshots  map[string][]*rds.DBSnapshot
	dbSnapshots          []*rds.DBSnapshot
	dbDeletingInstances  []string
	dbDeletedInstances   []string
}

// InstanceDeleting checks if instance ID was requested to be deleted.
func (m *mockRDSClient) InstanceDeleting(id string) bool {
	for _, deletingID := range m.dbDeletingInstances {
		if id == deletingID {
			return true
		}
	}
	return false
}

// InstanceDeleted checks if instance ID was effectively deleted.
func (m *mockRDSClient) InstanceDeleted(id string) bool {
	for _, deletedID := range m.dbDeletedInstances {
		if id == deletedID {
			return true
		}
	}
	return false
}

// DeleteDBInstance mocks rds.DeleteDBInstance.
func (m *mockRDSClient) DeleteDBInstance(
	params *rds.DeleteDBInstanceInput,
) (
	result *rds.DeleteDBInstanceOutput,
	err error,
) {
	id := params.DBInstanceIdentifier
	m.dbDeletingInstances = append(
		m.dbDeletingInstances,
		*id,
	)
	endpoint := fmt.Sprintf(
		"%s.amazonaws.com",
		id,
	)
	port := int64(5432)
	m.dbInstancesEndpoints[*id] = rds.Endpoint{
		Address: &endpoint,
		Port:    &port,
	}
	result = &rds.DeleteDBInstanceOutput{
		DBInstance: &rds.DBInstance{
			DBInstanceIdentifier: params.DBInstanceIdentifier,
			DBInstanceStatus:     aws.String("deleting"),
		},
	}
	return
}

// AddSnapshots add a list of snapshots to the mock, both in the full
// list and in the per instance map.
func (m *mockRDSClient) AddSnapshots(
	snapshots []*rds.DBSnapshot,
) {
	m.dbSnapshots = []*rds.DBSnapshot{}
	for _, snapshot := range snapshots {
		m.AddSnapshot(snapshot)
	}
}

// AddSnapshot add a new snapshot to the mock, both in the full list
// and the in the per instance map.
func (m *mockRDSClient) AddSnapshot(
	snapshot *rds.DBSnapshot,
) {
	m.dbSnapshots = append(m.dbSnapshots, snapshot)
	id := *snapshot.DBInstanceIdentifier
	if _, ok := m.dbInstanceSnapshots[id]; !ok {
		m.dbInstanceSnapshots[id] = []*rds.DBSnapshot{}
	}
	m.dbInstanceSnapshots[id] = append(m.dbInstanceSnapshots[id], snapshot)
}

// DescribeDBSnapshots mocks rds.DescribeDBSnapshots.
func (m mockRDSClient) DescribeDBSnapshots(
	describeParams *rds.DescribeDBSnapshotsInput,
) (
	result *rds.DescribeDBSnapshotsOutput,
	err error,
) {
	var snapshots []*rds.DBSnapshot
	if describeParams.DBInstanceIdentifier != nil {
		id := describeParams.DBInstanceIdentifier
		snapshots = m.dbInstanceSnapshots[*id]
	} else {
		snapshots = m.dbSnapshots
	}
	result = &rds.DescribeDBSnapshotsOutput{
		DBSnapshots: snapshots,
	}
	return
}

// DescribeDBInstances mocks rds.DescribeDBInstances.
func (m *mockRDSClient) DescribeDBInstances(
	describeParams *rds.DescribeDBInstancesInput,
) (
	result *rds.DescribeDBInstancesOutput,
	err error,
) {
	id := describeParams.DBInstanceIdentifier
	if m.InstanceDeleted(*id) {
		err = fmt.Errorf(
			"No such instance %s",
			id,
		)
		return
	}
	status := "available"
	if m.InstanceDeleting(*id) {
		status = "deleting"
		m.dbDeletedInstances = append(
			m.dbDeletedInstances,
			*id,
		)
	}
	endpoint, _ := m.dbInstancesEndpoints[*id]
	result = &rds.DescribeDBInstancesOutput{
		DBInstances: []*rds.DBInstance{
			{
				DBInstanceIdentifier: id,
				DBInstanceStatus:     &status,
				Endpoint:             &endpoint,
			},
		},
	}
	return
}

// CreateDBInstance mocks rds.CreateDBInstance.
func (m mockRDSClient) CreateDBInstance(
	inputParams *rds.CreateDBInstanceInput,
) (
	result *rds.CreateDBInstanceOutput,
	err error,
) {
	if err = inputParams.Validate(); err != nil {
		return
	}
	az := "us-east-1c"
	if inputParams.AvailabilityZone != nil {
		az = *inputParams.AvailabilityZone
	}
	if inputParams.MasterUsername == nil ||
		*inputParams.MasterUsername == "" {
		err = errors.New("Specify Master User")
		return
	}
	if inputParams.MasterUserPassword == nil ||
		*inputParams.MasterUserPassword == "" {
		err = errors.New("Specify Master User Password")
		return
	}
	if inputParams.AllocatedStorage == nil ||
		*inputParams.AllocatedStorage < 5 ||
		*inputParams.AllocatedStorage > 6144 {
		err = errors.New("Specify size between 5 and 6144")
		return
	}
	region := az[:len(az)-1]
	id := inputParams.DBInstanceIdentifier
	endpoint := fmt.Sprintf(
		"%s.0.%s.rds.amazonaws.com",
		*id,
		region,
	)
	port := int64(5432)
	m.dbInstancesEndpoints[*id] = rds.Endpoint{
		Address: &endpoint,
		Port:    &port,
	}
	status := "creating"
	result = &rds.CreateDBInstanceOutput{
		DBInstance: &rds.DBInstance{
			AllocatedStorage: inputParams.AllocatedStorage,
			DBInstanceArn: aws.String(
				fmt.Sprintf(
					"arn:aws:rds:%s:0:db:%s",
					region,
					id,
				),
			),
			DBInstanceIdentifier: id,
			DBInstanceStatus:     &status,
			Engine:               inputParams.Engine,
		},
	}
	return
}

// RestoreDBInstanceFromDBSnapshot mocks rds.RestoreDBInstanceFromDBSnapshot.
func (m mockRDSClient) RestoreDBInstanceFromDBSnapshot(
	inputParams *rds.RestoreDBInstanceFromDBSnapshotInput,
) (
	result *rds.RestoreDBInstanceFromDBSnapshotOutput,
	err error,
) {
	if err = inputParams.Validate(); err != nil {
		return
	}
	az := "us-east-1c"
	if inputParams.AvailabilityZone != nil {
		az = *inputParams.AvailabilityZone
	}
	region := az[:len(az)-1]
	id := inputParams.DBInstanceIdentifier
	endpoint := fmt.Sprintf(
		"%s.0.%s.rds.amazonaws.com",
		*id,
		region,
	)
	port := int64(5432)
	m.dbInstancesEndpoints[*id] = rds.Endpoint{
		Address: &endpoint,
		Port:    &port,
	}
	status := "creating"
	result = &rds.RestoreDBInstanceFromDBSnapshotOutput{
		DBInstance: &rds.DBInstance{
			DBInstanceArn: aws.String(
				fmt.Sprintf(
					"arn:aws:rds:%s:0:db:%s",
					region,
					id,
				),
			),
			DBInstanceIdentifier: id,
			DBInstanceStatus:     &status,
			Engine:               inputParams.Engine,
		},
	}
	return
}

// ModifyDBInstance mocks rds.ModifyDBInstance.
func (m mockRDSClient) ModifyDBInstance(
	inputParams *rds.ModifyDBInstanceInput,
) (
	result *rds.ModifyDBInstanceOutput,
	err error,
) {
	if err = inputParams.Validate(); err != nil {
		return
	}
	result = &rds.ModifyDBInstanceOutput{
		DBInstance: &rds.DBInstance{
			DBInstanceIdentifier: inputParams.DBInstanceIdentifier,
		},
	}
	return
}

// newMockRDSClient creates a mockRDSClient.
func newMockRDSClient() *mockRDSClient {
	return &mockRDSClient{
		dbInstancesEndpoints: map[string]rds.Endpoint{},
		dbInstanceSnapshots:  map[string][]*rds.DBSnapshot{},
		dbSnapshots:          []*rds.DBSnapshot{},
		dbDeletingInstances:  []string{},
		dbDeletedInstances:   []string{},
	}
}
