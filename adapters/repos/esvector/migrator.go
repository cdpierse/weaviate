package esvector

import (
	"context"
	"fmt"
	"strings"

	"github.com/semi-technologies/weaviate/entities/models"
	"github.com/semi-technologies/weaviate/entities/schema"
	"github.com/semi-technologies/weaviate/entities/schema/kind"
)

// Migrator is a wrapper around a "primitive" esvector.Repo which implements
// the migrate.Migrator interface
type Migrator struct {
	repo *Repo
}

// NewMigrator from esvector.Repo to implement migrate.Migrator interface
func NewMigrator(repo *Repo) *Migrator {
	return &Migrator{repo: repo}
}

// AddClass creates an index, then puts the desired mappings
func (m *Migrator) AddClass(ctx context.Context, kind kind.Kind, class *models.SemanticSchemaClass) error {
	index := classIndexFromClass(kind, class)
	err := m.repo.PutIndex(ctx, index)
	if err != nil {
		return fmt.Errorf("add class %s: create index: %v", class.Class, err)
	}

	err = m.setMappings(ctx, index, class.Properties)
	if err != nil {
		return fmt.Errorf("add class %s: map properties: %v", class.Class, err)
	}

	return nil
}

// DropClass deletes a class specific index
func (m *Migrator) DropClass(ctx context.Context, kind kind.Kind, className string) error {
	index := classIndexFromClassName(kind, className)
	err := m.repo.DeleteIndex(ctx, index)
	if err != nil {
		return fmt.Errorf("drop class %s: delete index: %v", className, err)
	}

	return nil
}

func (m *Migrator) UpdateClass(ctx context.Context, kind kind.Kind, className string, newClassName *string, newKeywords *models.SemanticSchemaKeywords) error {
	panic("not implemented")
}

func (m *Migrator) AddProperty(ctx context.Context, kind kind.Kind, className string, prop *models.SemanticSchemaClassProperty) error {
	// put mappings does not delete existing properties, so we can use it to add
	// a new one, too
	index := classIndexFromClassName(kind, className)
	err := m.setMappings(ctx, index, []*models.SemanticSchemaClassProperty{prop})
	if err != nil {
		return fmt.Errorf("add property %s to class %s: map properties: %v",
			className, prop.Name, err)
	}

	return nil
}

func (m *Migrator) DropProperty(ctx context.Context, kind kind.Kind, className string, propertyName string) error {
	panic("not implemented")
}

func (m *Migrator) UpdateProperty(ctx context.Context, kind kind.Kind, className string, propName string, newName *string, newKeywords *models.SemanticSchemaKeywords) error {
	panic("not implemented")
}

func (m *Migrator) UpdatePropertyAddDataType(ctx context.Context, kind kind.Kind, className string, propName string, newDataType string) error {
	panic("not implemented")
}

const indexPrefix = "class_"

func classIndexFromClass(kind kind.Kind, class *models.SemanticSchemaClass) string {
	return classIndexFromClassName(kind, class.Class)
}

func classIndexFromClassName(kind kind.Kind, className string) string {
	return fmt.Sprintf("%s%s_%s",
		indexPrefix, kind.Name(), strings.ToLower(className))
}

func (m *Migrator) setMappings(ctx context.Context, index string,
	props []*models.SemanticSchemaClassProperty) error {
	esProperties := map[string]interface{}{}

	for _, prop := range props {
		if len(prop.DataType) > 1 {
			// this must be a ref-type prop, so we can safely skip it
			continue
		}

		switch prop.DataType[0] {
		case string(schema.DataTypeString):
			esProperties[prop.Name] = typeMap(Keyword)
		case string(schema.DataTypeText):
			esProperties[prop.Name] = typeMap(Text)
		case string(schema.DataTypeInt):
			esProperties[prop.Name] = typeMap(Integer)
		case string(schema.DataTypeNumber):
			esProperties[prop.Name] = typeMap(Float)
		case string(schema.DataTypeBoolean):
			esProperties[prop.Name] = typeMap(Boolean)
		case string(schema.DataTypeDate):
			esProperties[prop.Name] = typeMap(Date)
		case string(schema.DataTypeGeoCoordinates):
			esProperties[prop.Name] = typeMap(GeoPoint)
		default:
			// assume everythings else must be a ref prop, simply ignore
			continue
		}
	}

	return m.repo.SetMappings(ctx, index, esProperties)
}

func typeMap(ft FieldType) map[string]interface{} {
	return map[string]interface{}{
		"type": ft,
	}
}