package mapper

import (
	"github.com/jinzhu/copier"
)

// CopierDBMapper is a reflection-based mapper using jinzhu/copier.
// Suitable for scenarios where field names align between domain and entity.
// D = Domain model (biz layer), E = Entity/PO (data layer).
type CopierDBMapper[D any, E any] struct {
	converters   []copier.TypeConverter
	options      copier.Option
	fieldMapping map[string]string
}

// NewCopierDBMapper creates a new CopierDBMapper instance.
func NewCopierDBMapper[D any, E any]() *CopierDBMapper[D, E] {
	return &CopierDBMapper[D, E]{
		converters:   make([]copier.TypeConverter, 0),
		fieldMapping: make(map[string]string),
		options: copier.Option{
			IgnoreEmpty: false,
			DeepCopy:    true,
		},
	}
}

func (m *CopierDBMapper[D, E]) WithIgnoreEmpty(ignore bool) *CopierDBMapper[D, E] {
	m.options.IgnoreEmpty = ignore
	return m
}

func (m *CopierDBMapper[D, E]) WithDeepCopy(deep bool) *CopierDBMapper[D, E] {
	m.options.DeepCopy = deep
	return m
}

// WithFieldMapping sets field name mappings (Domain field -> Entity field).
func (m *CopierDBMapper[D, E]) WithFieldMapping(mapping map[string]string) *CopierDBMapper[D, E] {
	for k, v := range mapping {
		m.fieldMapping[k] = v
	}
	m.options.FieldNameMapping = m.buildFieldNameMapping()
	return m
}

func (m *CopierDBMapper[D, E]) buildFieldNameMapping() []copier.FieldNameMapping {
	if len(m.fieldMapping) == 0 {
		return nil
	}
	mappings := make([]copier.FieldNameMapping, 0, len(m.fieldMapping)*2)
	for domainField, entityField := range m.fieldMapping {
		mappings = append(mappings, copier.FieldNameMapping{
			SrcType: new(D),
			DstType: new(E),
			Mapping: map[string]string{domainField: entityField},
		})
		mappings = append(mappings, copier.FieldNameMapping{
			SrcType: new(E),
			DstType: new(D),
			Mapping: map[string]string{entityField: domainField},
		})
	}
	return mappings
}

func (m *CopierDBMapper[D, E]) RegisterConverter(converter copier.TypeConverter) *CopierDBMapper[D, E] {
	m.converters = append(m.converters, converter)
	return m
}

func (m *CopierDBMapper[D, E]) RegisterConverters(converters []copier.TypeConverter) *CopierDBMapper[D, E] {
	m.converters = append(m.converters, converters...)
	return m
}

func (m *CopierDBMapper[D, E]) ToDomain(entity *E) *D {
	if entity == nil {
		return nil
	}
	var domain D
	opt := m.options
	opt.Converters = m.converters
	if err := copier.CopyWithOption(&domain, entity, opt); err != nil {
		return nil
	}
	return &domain
}

func (m *CopierDBMapper[D, E]) ToEntity(domain *D) *E {
	if domain == nil {
		return nil
	}
	var entity E
	opt := m.options
	opt.Converters = m.converters
	if err := copier.CopyWithOption(&entity, domain, opt); err != nil {
		return nil
	}
	return &entity
}

func (m *CopierDBMapper[D, E]) ToDomainList(entities []*E) []*D {
	if len(entities) == 0 {
		return nil
	}
	domains := make([]*D, 0, len(entities))
	for _, entity := range entities {
		if d := m.ToDomain(entity); d != nil {
			domains = append(domains, d)
		}
	}
	return domains
}

func (m *CopierDBMapper[D, E]) ToEntityList(domains []*D) []*E {
	if len(domains) == 0 {
		return nil
	}
	entities := make([]*E, 0, len(domains))
	for _, domain := range domains {
		if e := m.ToEntity(domain); e != nil {
			entities = append(entities, e)
		}
	}
	return entities
}
