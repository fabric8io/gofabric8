package rendering

// MarkupContent defines the raw content of a field along with the markup language used to input the content.
type MarkupContent struct {
	Content string `json:"content"`
	Markup  string `json:"markup"`
}

const (
	// the key for the 'content' field when the MarkupContent is converted into/from a Map
	ContentKey = "content"
	// the key for the 'markup' field when the MarkupContent is converted into/from a Map
	MarkupKey = "markup"
)

func (markupContent *MarkupContent) ToMap() map[string]interface{} {
	result := make(map[string]interface{})
	result[ContentKey] = markupContent.Content
	if markupContent.Markup == "" {
		result[MarkupKey] = SystemMarkupDefault
	} else {
		result[MarkupKey] = markupContent.Markup
	}
	return result
}

// NewMarkupContentFromMap creates a MarkupContent from the given Map,
// filling the 'Markup' field with the default value if no entry was found in the input or if the given markup is not supported.
// This avoids filling the DB with invalid markup types.
func NewMarkupContentFromMap(value map[string]interface{}) MarkupContent {
	content := value[ContentKey].(string)
	markup := SystemMarkupDefault
	if m, ok := value[MarkupKey]; ok {
		markup = m.(string)
		// use default markup if the input is not supported
		if !IsMarkupSupported(markup) {
			markup = SystemMarkupDefault
		}
	}
	return MarkupContent{Content: content, Markup: markup}
}

// NewMarkupContentFromLegacy creates a MarkupContent from the given content, using the default markup.
func NewMarkupContentFromLegacy(content string) MarkupContent {
	return MarkupContent{Content: content, Markup: SystemMarkupDefault}
}

// NewMarkupContent creates a MarkupContent from the given content, using the default markup.
func NewMarkupContent(content, markup string) MarkupContent {
	return MarkupContent{Content: content, Markup: markup}
}

// NewMarkupContentFromValue creates a MarkupContent from the given value,
// by converting a 'string', a 'map[string]interface{}' or casting a 'MarkupContent'. Otherwise, it returns nil.
func NewMarkupContentFromValue(value interface{}) *MarkupContent {
	if value == nil {
		return nil
	}
	switch value.(type) {
	case string:
		result := NewMarkupContentFromLegacy(value.(string))
		return &result
	case MarkupContent:
		result := value.(MarkupContent)
		return &result
	case map[string]interface{}:
		result := NewMarkupContentFromMap(value.(map[string]interface{}))
		return &result
	default:
		return nil
	}
}

// NilSafeGetMarkup returns the given markup if it is not nil nor empty, otherwise it returns the default markup
func NilSafeGetMarkup(markup *string) string {
	if markup != nil && *markup != "" {
		return *markup
	}
	return SystemMarkupDefault
}
