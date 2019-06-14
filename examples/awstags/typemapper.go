// +build typemapper

package awstags

import (
	"github.com/aws/aws-sdk-go/service/acm"
	"github.com/aws/aws-sdk-go/service/datasync"
	"github.com/aws/aws-sdk-go/service/directoryservice"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elbv2"

	"github.com/paultyng/go-typemapper"
)

func (src *tag) ACMTag() *acm.Tag {
	dst := &acm.Tag{}
	typemapper.CreateMap(src, dst)
	return dst
}

func (src tags) ACMTags() []*acm.Tag {
	var dst []*acm.Tag
	typemapper.CreateMap(src, dst)
	typemapper.MapWith(src[0].ACMTag)
	return dst
}

func (src *tag) DataSyncTag() *datasync.TagListEntry {
	var dst *datasync.TagListEntry
	typemapper.CreateMap(src, dst)
	return dst
}

func (src tags) DataSyncTags() []*datasync.TagListEntry {
	var dst []*datasync.TagListEntry
	typemapper.CreateMap(src, dst)
	typemapper.MapWith(src[0].DataSyncTag)
	return dst
}

func (src *tag) DirectoryServiceTag() *directoryservice.Tag {
	dst := &directoryservice.Tag{}
	typemapper.CreateMap(src, dst)
	return dst
}

func (src tags) DirectoryServiceTags() []*directoryservice.Tag {
	var dst []*directoryservice.Tag
	typemapper.CreateMap(src, dst)
	typemapper.MapWith(src[0].DirectoryServiceTag)
	return dst
}

func (src *tag) EC2Tag() *ec2.Tag {
	dst := &ec2.Tag{}
	typemapper.CreateMap(src, dst)
	return dst
}

func (src tags) EC2Tags() []*ec2.Tag {
	var dst []*ec2.Tag
	typemapper.CreateMap(src, dst)
	typemapper.MapWith(src[0].EC2Tag)
	return dst
}

func (src *tag) ELBV2Tag() *elbv2.Tag {
	dst := &elbv2.Tag{}
	typemapper.CreateMap(src, dst)
	return dst
}

func (src tags) ELBV2Tags() []*elbv2.Tag {
	var dst []*elbv2.Tag
	typemapper.CreateMap(src, dst)
	typemapper.MapWith(src[0].ELBV2Tag)
	return dst
}
