// VIBED ALL THE AY LUL
package anime

import (
	"context"
	"encoding/json"
	"log"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/jomei/notionapi"
)

// dateOnly is a notionapi.Property that serializes its date as YYYY-MM-DD,
// omitting any time component. Used in place of *notionapi.DateProperty so
// Notion treats the value as a date (not a datetime). The lib's *Date type
// always marshals via RFC3339, so we substitute a custom marshaler.
type dateOnly struct {
	Type  notionapi.PropertyType
	Start time.Time
}

func (d *dateOnly) GetID() string                   { return "" }
func (d *dateOnly) GetType() notionapi.PropertyType { return d.Type }

func (d *dateOnly) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type notionapi.PropertyType `json:"type,omitempty"`
		Date struct {
			Start string `json:"start"`
		} `json:"date"`
	}{
		Type: d.Type,
		Date: struct {
			Start string `json:"start"`
		}{Start: d.Start.Format("2006-01-02")},
	})
}

const SLUG_COLUMN = "Slug"

func loadAnimesFromNotion(client *notionapi.Client) (map[string]Status, error) {
	req := notionapi.DatabaseQueryRequest{}

	out := []notionapi.Page{}
	for {
		resp, err := client.Database.Query(context.Background(), ANIME_DATABASE, &req)
		if err != nil {
			return nil, err
		}

		out = append(out, resp.Results...)
		if !resp.HasMore {
			break
		}

		req.StartCursor = resp.NextCursor
	}

	result := map[string]Status{}
	for _, v := range out {
		stat := Status{}
		unmarshalPage(v, &stat)

		if stat.Name == "" {
			continue
		}

		result[stat.Name] = stat
	}

	return result, nil
}

// unmarshalPage populates the struct pointed to by dst from a Notion page.
//
// Each exported field's source is chosen by the `notion:"<property>"` struct
// tag, falling back to the Go field name. The special name "[Name]" resolves
// to the page's title (the one TitleProperty in Properties) regardless of the
// column it lives in; every other name is a direct Properties lookup. Missing
// properties leave the field at its zero value.
func unmarshalPage(page notionapi.Page, dst any) {
	v := reflect.ValueOf(dst)
	if v.Kind() != reflect.Pointer || v.IsNil() || v.Elem().Kind() != reflect.Struct {
		log.Fatalf("unmarshalPage: dst must be a non-nil pointer to a struct, got %T", dst)
	}
	v = v.Elem()

	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		name := field.Tag.Get("notion")
		if name == "" {
			name = field.Name
		}
		if name == "-" {
			continue
		}

		var prop notionapi.Property
		if name == "[Name]" {
			prop = findTitleProperty(page.Properties)
		} else {
			prop = page.Properties[name]
		}
		if prop == nil {
			continue
		}

		setFromProperty(v.Field(i), prop)
	}
}

func findTitleProperty(props notionapi.Properties) notionapi.Property {
	for _, p := range props {
		if p.GetType() == notionapi.PropertyTypeTitle {
			return p
		}
	}
	return nil
}

func setFromProperty(field reflect.Value, prop notionapi.Property) {
	switch p := prop.(type) {
	case *notionapi.TitleProperty:
		setString(field, joinRichText(p.Title))
	case *notionapi.RichTextProperty:
		setString(field, joinRichText(p.RichText))
	case *notionapi.NumberProperty:
		setNumber(field, p.Number)
	case *notionapi.URLProperty:
		setString(field, p.URL)
	case *notionapi.SelectProperty:
		setString(field, p.Select.Name)
	case *notionapi.StatusProperty:
		setString(field, p.Status.Name)
	case *notionapi.DateProperty:
		if p.Date != nil && p.Date.Start != nil {
			setTime(field, time.Time(*p.Date.Start))
		}
	case *notionapi.CreatedTimeProperty:
		setTime(field, p.CreatedTime)
	case *notionapi.LastEditedTimeProperty:
		setTime(field, p.LastEditedTime)
	}
}

func setString(field reflect.Value, s string) {
	switch field.Kind() {
	case reflect.String:
		field.SetString(s)
	case reflect.Int, reflect.Int64:
		n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
		if err == nil {
			field.SetInt(n)
		}
	}
}

func setNumber(field reflect.Value, n float64) {
	if field.Kind() == reflect.Int || field.Kind() == reflect.Int64 {
		field.SetInt(int64(n))
	}
}

func setTime(field reflect.Value, t time.Time) {
	if field.Type() == reflect.TypeOf(time.Time{}) {
		field.Set(reflect.ValueOf(t))
	}
}

func joinRichText(rt []notionapi.RichText) string {
	parts := make([]string, 0, len(rt))
	for _, t := range rt {
		parts = append(parts, t.PlainText)
	}
	return strings.Join(parts, "")
}

// saveAll re-queries the Notion database, matches each local Status to a page
// by SLUG_COLUMN, and updates the writable properties on the matched page.
// Duplicate slugs are fatal. Local statuses without a Notion page are logged
// and skipped. Only properties already present in the page's Properties bag
// are written; read-only/derived property types are skipped.
func saveAll(client *notionapi.Client, animes map[string]Status) error {
	req := notionapi.DatabaseQueryRequest{}
	pages := []notionapi.Page{}
	for {
		resp, err := client.Database.Query(context.Background(), ANIME_DATABASE, &req)
		if err != nil {
			return err
		}
		pages = append(pages, resp.Results...)
		if !resp.HasMore {
			break
		}
		req.StartCursor = resp.NextCursor
	}

	bySlug := map[string]notionapi.Page{}
	for _, p := range pages {
		slug := propertyToString(p.Properties[SLUG_COLUMN])
		if slug == "" {
			continue
		}
		existing, dup := bySlug[slug]
		if dup {
			log.Fatalf("save: duplicate slug %q on pages %s and %s", slug, existing.ID, p.ID)
		}
		bySlug[slug] = p
	}

	for _, status := range animes {
		slug := status.Name
		page, found := bySlug[slug]
		if !found {
			log.Printf("save: no Notion page for slug %q, skipping", slug)
			continue
		}
		props := buildUpdate(page.Properties, status)
		if len(props) == 0 {
			continue
		}

		_, err := client.Page.Update(context.Background(), notionapi.PageID(page.ID), &notionapi.PageUpdateRequest{
			Properties: props,
		})
		if err != nil {
			log.Printf("save: update slug %q failed: %v", slug, err)
		}
	}

	return nil
}

// propertyToString extracts a string-ish value from any string-shaped property.
// Used to read the slug regardless of which Notion column type it lives in.
func propertyToString(prop notionapi.Property) string {
	switch p := prop.(type) {
	case *notionapi.TitleProperty:
		return joinRichText(p.Title)
	case *notionapi.RichTextProperty:
		return joinRichText(p.RichText)
	case *notionapi.URLProperty:
		return p.URL
	case *notionapi.SelectProperty:
		return p.Select.Name
	case *notionapi.StatusProperty:
		return p.Status.Name
	case *notionapi.NumberProperty:
		return strconv.FormatFloat(p.Number, 'f', -1, 64)
	}
	return ""
}

func buildUpdate(orig notionapi.Properties, src any) notionapi.Properties {
	fields := fieldMap(src)
	out := notionapi.Properties{}
	for name, prop := range orig {
		field, ok := fields[name]
		if !ok {
			continue
		}
		newProp := fieldToProperty(prop, field)
		if newProp != nil {
			out[name] = newProp
		}
	}
	return out
}

func fieldMap(src any) map[string]reflect.Value {
	v := reflect.ValueOf(src)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	out := map[string]reflect.Value{}
	if v.Kind() != reflect.Struct {
		return out
	}
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		name := f.Tag.Get("notion")
		if name == "" {
			name = f.Name
		}
		if name == "-" {
			continue
		}
		out[name] = v.Field(i)
	}
	return out
}

// fieldToProperty builds a writable Notion property of the same concrete type
// as prop, carrying field's value. Returns nil to skip (empty string, zero
// number, zero time) or for read-only/derived property types.
func fieldToProperty(prop notionapi.Property, field reflect.Value) notionapi.Property {
	switch p := prop.(type) {
	case *notionapi.RichTextProperty:
		s := fieldToString(field)
		if s == "" {
			return nil
		}
		return &notionapi.RichTextProperty{
			Type: p.Type,
			RichText: []notionapi.RichText{{
				Type: "text",
				Text: &notionapi.Text{Content: s},
			}},
		}
	case *notionapi.NumberProperty:
		f := fieldToFloat(field)
		if f == 0 {
			return nil
		}
		return &notionapi.NumberProperty{Type: p.Type, Number: f}
	case *notionapi.URLProperty:
		s := fieldToString(field)
		if s == "" {
			return nil
		}
		return &notionapi.URLProperty{Type: p.Type, URL: s}
	case *notionapi.SelectProperty:
		s := fieldToString(field)
		if s == "" {
			return nil
		}
		return &notionapi.SelectProperty{
			Type:   p.Type,
			Select: notionapi.Option{Name: s},
		}
	case *notionapi.DateProperty:
		if field.Type() != reflect.TypeOf(time.Time{}) {
			return nil
		}
		t := field.Interface().(time.Time)
		if t.IsZero() {
			return nil
		}
		return &dateOnly{Type: p.Type, Start: t}
	}
	return nil
}

func fieldToString(field reflect.Value) string {
	if field.Type() == reflect.TypeOf(time.Time{}) {
		t := field.Interface().(time.Time)
		if t.Year() < 2000 {
			return ""
		}

		return t.Format("2006-01-02")
	}
	switch field.Kind() {
	case reflect.String:
		return field.String()
	case reflect.Int, reflect.Int64:
		return strconv.FormatInt(field.Int(), 10)
	}
	return ""
}

func fieldToFloat(field reflect.Value) float64 {
	if field.Kind() != reflect.Int && field.Kind() != reflect.Int64 {
		return 0
	}
	return float64(field.Int())
}
