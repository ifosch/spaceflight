package odin

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/rds/rdsiface"
)

// Init initializes connection to AWS API
func Init() rdsiface.RDSAPI {
	region := "us-east-1"
	sess := session.New(&aws.Config{Region: aws.String(region)})
	return rds.New(sess)
}

var duration = time.Duration(5) * time.Second

// CreateDBParams represents CreateDBInstance parameters.
type CreateDBParams struct {
	DBInstanceType string
	DBUser         string
	DBPassword     string
	Size           int64

	OriginalInstanceName string
}

// GetCreateDBInstanceInput method creates a new CreateDBInstanceInput from provided
// CreateDBParams and rds.DBSnapshot.
func (params CreateDBParams) GetCreateDBInstanceInput(
	identifier string,
	svc rdsiface.RDSAPI,
) (
	createDBInstanceInput *rds.CreateDBInstanceInput,
	err error,
) {
	var snapshot *rds.DBSnapshot
	if params.OriginalInstanceName != "" {
		snapshot, err = GetLastSnapshot(params.OriginalInstanceName, svc)
		if err != nil {
			return
		}
	}
	createDBInstanceInput = &rds.CreateDBInstanceInput{
		AllocatedStorage:     &params.Size,
		DBInstanceIdentifier: &identifier,
		DBInstanceClass:      &params.DBInstanceType,
		DBSecurityGroups: []*string{
			aws.String("default"),
		},
		Engine:             aws.String("postgres"),
		EngineVersion:      aws.String("9.4.11"),
		MasterUsername:     &params.DBUser,
		MasterUserPassword: &params.DBPassword,
		Tags: []*rds.Tag{
			{
				Key:   aws.String("Name"),
				Value: &identifier,
			},
		},
	}
	if snapshot != nil {
		createDBInstanceInput.AllocatedStorage = snapshot.AllocatedStorage
		createDBInstanceInput.MasterUsername = snapshot.MasterUsername
	}
	if err = createDBInstanceInput.Validate(); err != nil {
		return
	}
	return
}

// CreateDBInstance creates a new RDS database instance. If a vpcid is
// specified the security group will be in that VPC.
func CreateDBInstance(
	instanceName string,
	params CreateDBParams,
	svc rdsiface.RDSAPI,
) (result string, err error) {
	rdsParams, err := params.GetCreateDBInstanceInput(
		instanceName,
		svc,
	)
	if err != nil {
		return
	}
	res, err := svc.CreateDBInstance(rdsParams)
	if err != nil {
		return
	}
	instance := *res.DBInstance
	for *instance.DBInstanceStatus != "available" {
		res2, err2 := svc.DescribeDBInstances(&rds.DescribeDBInstancesInput{
			DBInstanceIdentifier: instance.DBInstanceIdentifier,
		})
		if err2 != nil {
			err = err2
			return
		}
		instance = *res2.DBInstances[0]
		// This is to avoid AWS API rate throttling.
		time.Sleep(duration)
	}
	result = *instance.Endpoint.Address
	return
}

// GetLastSnapshot queries AWS looking for a Snapshot ID, depending on
// an instance ID.
func GetLastSnapshot(
	identifier string,
	svc rdsiface.RDSAPI,
) (result *rds.DBSnapshot, err error) {
	params := &rds.DescribeDBSnapshotsInput{
		DBInstanceIdentifier: &identifier,
	}
	results, err := svc.DescribeDBSnapshots(params)
	if err != nil {
		return
	}
	result = results.DBSnapshots[0]
	return
}