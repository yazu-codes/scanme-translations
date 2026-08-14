package services

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/yazu-codes/scanme-translations.git/src/dto"
	"golang.org/x/text/language"

	"cloud.google.com/go/translate"
)

type TranslationService struct {
	translateClient *translate.Client
	ctx             context.Context
}

type Chunk struct {
	text  string
	start int // which field index this chunk starts at
	count int // how many fields in this chunk
}

type CategoryNameMapping struct {
	Source string `json: "source"`
	Target string `json: "target"`
}

type CategoryNameMappings []CategoryNameMapping

// FindTarget looks up the target (translated) value for a given source string.
// Returns the target and true if found, or empty string and false if not found.
func (c CategoryNameMappings) FindTarget(source string) string {
	for _, m := range c {
		if m.Source == source {
			return m.Target
		}
	}
	return ""
}

// extractJSONStrings scans raw JSON text char-by-char, finds every string
// literal, and replaces any value that isn't "label" or "children" with a
// placeholder like "{0}", "{1}", etc. Returns the templated JSON (still
// valid JSON syntax) and the list of extracted strings in order.
func extractJSONStrings(jsonStr string) (template string, extracted []string) {
	var sb strings.Builder
	i := 0
	n := len(jsonStr)
	placeholderIdx := 0

	for i < n {
		c := jsonStr[i]
		if c == '"' {
			start := i
			i++
			for i < n {
				if jsonStr[i] == '\\' && i+1 < n {
					i += 2 // skip escaped char (e.g. \", \\, \n)
					continue
				}
				if jsonStr[i] == '"' {
					i++
					break
				}
				i++
			}
			raw := jsonStr[start:i]        // includes quotes
			content := raw[1 : len(raw)-1] // strip quotes

			if content == "label" || content == "children" {
				sb.WriteString(raw) // keep keys untouched
			} else {
				sb.WriteString(fmt.Sprintf("\"{%d}\"", placeholderIdx))
				extracted = append(extracted, content)
				placeholderIdx++
			}
		} else {
			sb.WriteByte(c)
			i++
		}
	}

	return sb.String(), extracted
}

// rebuildJSON substitutes each placeholder with its translated value,
// properly quoted/escaped for JSON.
func rebuildJSON(template string, translated []string) string {
	result := template
	for idx, val := range translated {
		placeholder := fmt.Sprintf("\"{%d}\"", idx)
		escaped := strconv.Quote(val) // handles quotes, backslashes, unicode
		result = strings.Replace(result, placeholder, escaped, 1)
	}
	return result
}

const delimiter = "<SPLIT/>"

func (s *TranslationService) translateCategoryOrder(categoryOrderJSON string, targetLang language.Tag) (string, CategoryNameMappings, error) {
	const delimiter = "<SPLIT/>"

	template, extracted := extractJSONStrings(categoryOrderJSON)

	if len(extracted) == 0 {
		return categoryOrderJSON, nil, nil // nothing to translate
	}

	text := strings.Join(extracted, delimiter)

	result, err := s.translateClient.Translate(
		s.ctx,
		[]string{text},
		targetLang,
		&translate.Options{Format: translate.Text},
	)
	if err != nil {
		return "", nil, err
	}

	translatedParts := strings.Split(result[0].Text, delimiter)
	if len(translatedParts) != len(extracted) {
		return "", nil, fmt.Errorf("translation split mismatch: got %d parts, expected %d", len(translatedParts), len(extracted))
	}

	// Build the source -> target mapping
	cnm := make([]CategoryNameMapping, 0, len(extracted))
	for i, source := range extracted {
		cnm = append(cnm, CategoryNameMapping{
			Source: source,
			Target: translatedParts[i],
		})
	}

	translatedJSON := rebuildJSON(template, translatedParts)

	return translatedJSON, cnm, nil
}

// func (s *TranslationService) splitText(text string, maxChunkSize int, delimiter string) []Chunk {
// 	parts := strings.Split(text, delimiter)

// 	var chunks []Chunk
// 	var currentChunk strings.Builder
// 	var startIdx int
// 	var fieldCount int
// 	var currentSize int

// 	for i, part := range parts {
// 		partSize := len(part) + len(delimiter)

// 		// If adding this part exceeds limit and we have fields, save chunk
// 		if currentSize+partSize > maxChunkSize && fieldCount > 0 {
// 			chunks = append(chunks, Chunk{
// 				text:  currentChunk.String(),
// 				start: startIdx,
// 				count: fieldCount,
// 			})
// 			currentChunk.Reset()
// 			startIdx = i
// 			fieldCount = 0
// 			currentSize = 0
// 		}

// 		if currentChunk.Len() > 0 {
// 			currentChunk.WriteString(delimiter)
// 		}
// 		currentChunk.WriteString(part)
// 		currentSize += partSize
// 		fieldCount++
// 	}

// 	// Add remaining chunk
// 	if fieldCount > 0 {
// 		chunks = append(chunks, Chunk{
// 			text:  currentChunk.String(),
// 			start: startIdx,
// 			count: fieldCount,
// 		})
// 	}

// 	return chunks
// }

func initTranslateClient(credentialsJSON string, ctx context.Context) *translate.Client {
	// Create tmp directory if it doesn't exist
	err := os.MkdirAll("tmp", 0755)
	if err != nil {
		log.Fatalf("Failed to create tmp directory: %v", err)
	}

	tempFile := "tmp/google-credentials.json"
	err = ioutil.WriteFile(tempFile, []byte(credentialsJSON), 0600)
	if err != nil {
		log.Fatalf("Failed to write credentials file: %v", err)
	}
	log.Printf("✅ Credentials written to %s", tempFile)

	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", tempFile)

	client, err := translate.NewClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create translate client: %v", err)
	}

	log.Println("✅ Google Translate client initialized")
	return client
}

func NewTranslationService(googleCreds string) *TranslationService {
	// Google Translate (uses GOOGLE_APPLICATION_CREDENTIALS env var)
	ctx := context.Background()
	translateClient := initTranslateClient(googleCreds, ctx)

	return &TranslationService{translateClient: translateClient, ctx: ctx}
}

func (s *TranslationService) Translate(menuDto dto.PublicMenu, sourceLanguage, targetLanguage string) (*dto.PublicMenu, error) {
	// fmt.Println("TRANSLATING MENU BEGIN:", menuDto)
	// fmt.Println("TRANSLATING TO LANGUAGE:", targetLanguage)

	// TODO: menuDTO owner information and menu item names, description, allergens should be converted to a string
	// representation, translated by google translate api and then parsed back into the menu owner info, item names,
	// descriptions and allergens fields.
	// Direct replacement, not introducing new fields.
	translatedMenu := ""
	const maxChunkSize = 2000
	const delayBetweenChunks = 200 * time.Millisecond // Safe buffer

	parsedLanguageTag, err := language.Parse(targetLanguage)
	if err != nil {
		return nil, err
	}

	translatedCategoryOrder, categoryMapping, err := s.translateCategoryOrder(menuDto.MenuConfiguration.CategoryOrder, parsedLanguageTag)
	if err != nil {
		return nil, err
	}

	menuDto.MenuConfiguration.CategoryOrder = translatedCategoryOrder

	translationLength := len(menuDto.MenuOwner.Name) + len(menuDto.MenuOwner.Slogan) // + len(menuDto.MenuConfiguration.CategoryOrder)

	menuStringRep := []string{}

	if len(menuDto.MenuOwner.Name) > 0 {
		menuStringRep = append(menuStringRep, menuDto.MenuOwner.Name)
	} else {
		menuStringRep = append(menuStringRep, "no name")
	}

	if len(menuDto.MenuOwner.Slogan) > 0 {
		menuStringRep = append(menuStringRep, menuDto.MenuOwner.Slogan)
	} else {
		menuStringRep = append(menuStringRep, "no slogan")
	}
	// menuStringRep = append(menuStringRep, menuDto.MenuConfiguration.CategoryOrder)

	for i, item := range menuDto.MenuItems {

		if len(item.Name) > 0 {
			menuStringRep = append(menuStringRep, item.Name)
		} else {
			menuStringRep = append(menuStringRep, "no name")
		}

		if len(item.Description) > 0 {
			menuStringRep = append(menuStringRep, item.Description)
		} else {
			menuStringRep = append(menuStringRep, "no description")
		}

		if len(item.Allergens) > 0 {
			menuStringRep = append(menuStringRep, item.Allergens)
		} else {
			menuStringRep = append(menuStringRep, "no allergens")
		}

		// if len(item.Category) > 0 {
		// 	menuStringRep = append(menuStringRep, item.Category)
		// } else {
		// 	menuStringRep = append(menuStringRep, "no category")
		// }

		translationLength = translationLength + len(item.Name) + len(item.Description) + len(item.Allergens) + len(item.Category)

		if translationLength > maxChunkSize || i == len(menuDto.MenuItems)-1 {
			text := strings.Join(menuStringRep, "<e/>")

			// fmt.Println("MENU STRING REP LENGTH:", len(text))
			// fmt.Println(text)

			translatedStringRep, err := s.translateClient.Translate(s.ctx, []string{text}, parsedLanguageTag, nil)
			if err != nil {
				return nil, err
			}

			// fmt.Println("TRANSLATED:", translatedStringRep[0].Text)

			translatedMenu += translatedStringRep[0].Text
			if i != len(menuDto.MenuItems)-1 {
				translatedMenu += "<e/>"
			}

			menuStringRep = []string{}

			translationLength = 0

			time.Sleep(delayBetweenChunks)
		}
	}

	// text := strings.Join(menuStringRep, "<e/>")

	// fmt.Println("MENU STRING REP LENGTH:", len(text))

	// translatedStringRep, err := s.translateClient.Translate(s.ctx, []string{text}, parsedLanguageTag, nil)
	// if err != nil {
	// 	return nil, err
	// }

	// fmt.Println("TRANSLATED:", translatedStringRep[0].Text)

	translatedStringParts := strings.Split(translatedMenu, "<e/>")

	menuDto.MenuOwner.Name = translatedStringParts[0]
	menuDto.MenuOwner.Slogan = translatedStringParts[1]
	// menuDto.MenuConfiguration.CategoryOrder = translatedStringParts[2]

	// fmt.Println(strings.Join(translatedStringParts, "\n"))

	translatedIndex := 2
	for i, _ := range menuDto.MenuItems {
		menuDto.MenuItems[i].Name = translatedStringParts[translatedIndex]
		menuDto.MenuItems[i].Description = translatedStringParts[translatedIndex+1]
		menuDto.MenuItems[i].Allergens = translatedStringParts[translatedIndex+2]
		menuDto.MenuItems[i].Category = categoryMapping.FindTarget(menuDto.MenuItems[i].Category)
		// fmt.Println("Translated item name:", menuDto.MenuItems[i].Name)
		// fmt.Println("Translated item desc:", menuDto.MenuItems[i].Description)
		// fmt.Println("Translated item allergens:", menuDto.MenuItems[i].Allergens)
		// fmt.Println("Translated item category:", menuDto.MenuItems[i].Category)
		translatedIndex += 3
	}

	return &menuDto, nil
}
