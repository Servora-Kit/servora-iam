package mapper

import (
	"github.com/jinzhu/copier"
)

// CopierProtoMapper is a reflection-based mapper for Protobuf messages and domain models.
// P = Protobuf message (API layer), D = Domain model (biz layer).
type CopierProtoMapper[P any, D any] struct {
	converters []copier.TypeConverter
	options    copier.Option
}

// NewCopierProtoMapper creates a new CopierProtoMapper with default timestamp converters.
func NewCopierProtoMapper[P any, D any]() *CopierProtoMapper[P, D] {
	m := &CopierProtoMapper[P, D]{
		converters: make([]copier.TypeConverter, 0),
		options: copier.Option{
			IgnoreEmpty: false,
			DeepCopy:    true,
		},
	}
	m.converters = append(m.converters, NewTimestamppbConverterPair()...)
	m.converters = append(m.converters, NewStringPointerConverterPair()...)
	m.converters = append(m.converters, NewInt64PointerConverterPair()...)
	return m
}

func (m *CopierProtoMapper[P, D]) RegisterConverter(converter copier.TypeConverter) *CopierProtoMapper[P, D] {
	m.converters = append(m.converters, converter)
	return m
}

func (m *CopierProtoMapper[P, D]) RegisterConverters(converters []copier.TypeConverter) *CopierProtoMapper[P, D] {
	m.converters = append(m.converters, converters...)
	return m
}

func (m *CopierProtoMapper[P, D]) ToDomain(proto *P) *D {
	if proto == nil {
		return nil
	}
	var domain D
	opt := m.options
	opt.Converters = m.converters
	if err := copier.CopyWithOption(&domain, proto, opt); err != nil {
		return nil
	}
	return &domain
}

func (m *CopierProtoMapper[P, D]) ToProto(domain *D) *P {
	if domain == nil {
		return nil
	}
	var proto P
	opt := m.options
	opt.Converters = m.converters
	if err := copier.CopyWithOption(&proto, domain, opt); err != nil {
		return nil
	}
	return &proto
}

func (m *CopierProtoMapper[P, D]) ToDomainList(protos []*P) []*D {
	if len(protos) == 0 {
		return nil
	}
	domains := make([]*D, 0, len(protos))
	for _, proto := range protos {
		if d := m.ToDomain(proto); d != nil {
			domains = append(domains, d)
		}
	}
	return domains
}

func (m *CopierProtoMapper[P, D]) ToProtoList(domains []*D) []*P {
	if len(domains) == 0 {
		return nil
	}
	protos := make([]*P, 0, len(domains))
	for _, domain := range domains {
		if p := m.ToProto(domain); p != nil {
			protos = append(protos, p)
		}
	}
	return protos
}
