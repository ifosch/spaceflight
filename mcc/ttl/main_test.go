package main

import (
	"github.com/aws/aws-sdk-go/service/route53"
	"testing"
)

var one = "one.example.com"
var two = "two.example.com"

var A = "A"
var AAAA = "AAAA"

var onerecordA = &route53.ResourceRecordSet{
	Name: &one,
	Type: &A,
}

var tworecordAAAA = &route53.ResourceRecordSet{
	Name: &two,
	Type: &AAAA,
}

var ResourceRecordSetList = []*route53.ResourceRecordSet{
	onerecordA,
	tworecordAAAA,
}

var rrstest = []struct {
	rrsl   []*route53.ResourceRecordSet
	filter []string
	out    []*route53.ResourceRecordSet
}{
	{
		ResourceRecordSetList,
		[]string{"A"},
		[]*route53.ResourceRecordSet{
			onerecordA,
		},
	},
	{
		ResourceRecordSetList,
		[]string{"AAAA"},
		[]*route53.ResourceRecordSet{
			tworecordAAAA,
		},
	},
	{
		ResourceRecordSetList,
		[]string{"MX"},
		[]*route53.ResourceRecordSet{},
	},
	{
		ResourceRecordSetList,
		[]string{"TXT"},
		[]*route53.ResourceRecordSet{},
	},
	{
		ResourceRecordSetList,
		[]string{
			"A",
			"AAAA",
		},
		[]*route53.ResourceRecordSet{
			onerecordA,
			tworecordAAAA,
		},
	},
}

func TestFilterResourceRecordSetType(t *testing.T) {
	for _, tt := range rrstest {
		name := tt.filter[0]
		for i := 1; i < len(tt.filter); i++ {
			name += "," + tt.filter[i]
		}
		t.Run(name, func(t *testing.T) {
			r := FilterResourceRecordSetType(tt.rrsl, tt.filter)
			if len(r) != len(tt.out) {
				t.Error("Result has different length than expected")
			}
			for index, value := range r {
				if tt.out[index] != value {
					t.Error("Results don't match as expected")
				}
			}
		})
	}
}