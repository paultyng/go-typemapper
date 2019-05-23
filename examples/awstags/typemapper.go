// +build typemapper

package awstags

import (
	"github.com/aws/aws-sdk-go/service/datasync"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elbv2"

	"github.com/paultyng/go-typemapper"
)

func (src *myTag) DataSyncTag(dst *datasync.TagListEntry) error {
	typemapper.CreateMap(src, dst)
	return nil
}

func EC2TagToDataSyncTag(src *ec2.Tag, dst *datasync.TagListEntry) error {
	typemapper.CreateMap(src, dst)
	return nil
}

func (src *myTag) EC2Tag() (*ec2.Tag, error) {
	dst := &ec2.Tag{}
	typemapper.CreateMap(src, dst)
	return dst, nil
}

func ELBv2TagToEC2Tag(src *elbv2.Tag, dst *ec2.Tag) error {
	typemapper.CreateMap(src, dst)
	return nil
}
