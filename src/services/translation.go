package services

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

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
	fmt.Println("TRANSLATING MENU BEGIN:", menuDto)

	// TODO: menuDTO owner information and menu item names, description, allergens should be converted to a string
	// representation, translated by google translate api and then parsed back into the menu owner info, item names,
	// descriptions and allergens fields.
	// Direct replacement, not introducing new fields.
	translatedMenu := ""
	const maxChunkSize = 3000

	parsedLanguageTag, err := language.Parse(targetLanguage)
	if err != nil {
		return nil, err
	}

	translationLength := len(menuDto.MenuOwner.Name) + len(menuDto.MenuOwner.Slogan) // + len(menuDto.MenuConfiguration.CategoryOrder)

	menuStringRep := []string{}
	menuStringRep = append(menuStringRep, menuDto.MenuOwner.Name)
	menuStringRep = append(menuStringRep, menuDto.MenuOwner.Slogan)
	// menuStringRep = append(menuStringRep, menuDto.MenuConfiguration.CategoryOrder)

	for i, item := range menuDto.MenuItems {
		menuStringRep = append(menuStringRep, item.Name)
		menuStringRep = append(menuStringRep, item.Description)
		menuStringRep = append(menuStringRep, item.Allergens)
		menuStringRep = append(menuStringRep, item.Category)

		translationLength = translationLength + len(item.Name) + len(item.Description) + len(item.Allergens) + len(item.Category)

		if translationLength > maxChunkSize || i == len(menuDto.MenuItems)-1 {
			text := strings.Join(menuStringRep, "{0}")

			fmt.Println("MENU STRING REP LENGTH:", len(text))
			fmt.Println(text)

			translatedStringRep, err := s.translateClient.Translate(s.ctx, []string{text}, parsedLanguageTag, nil)
			if err != nil {
				return nil, err
			}

			fmt.Println("TRANSLATED:", translatedStringRep[0].Text)

			translatedMenu += translatedStringRep[0].Text

			menuStringRep = []string{}
		}
	}

	// text := strings.Join(menuStringRep, "{0}")

	// fmt.Println("MENU STRING REP LENGTH:", len(text))

	// translatedStringRep, err := s.translateClient.Translate(s.ctx, []string{text}, parsedLanguageTag, nil)
	// if err != nil {
	// 	return nil, err
	// }

	// fmt.Println("TRANSLATED:", translatedStringRep[0].Text)

	translatedStringParts := strings.Split(translatedMenu, "{0}")

	menuDto.MenuOwner.Name = translatedStringParts[0]
	menuDto.MenuOwner.Slogan = translatedStringParts[1]
	// menuDto.MenuConfiguration.CategoryOrder = translatedStringParts[2]

	fmt.Println(translatedStringParts[3])

	translatedIndex := 3
	for i, _ := range menuDto.MenuItems {
		menuDto.MenuItems[i].Name = translatedStringParts[translatedIndex]
		menuDto.MenuItems[i].Description = translatedStringParts[translatedIndex+1]
		menuDto.MenuItems[i].Allergens = translatedStringParts[translatedIndex+2]
		menuDto.MenuItems[i].Category = translatedStringParts[translatedIndex+3]
		fmt.Println("Translated item:", menuDto.MenuItems[i])
		translatedIndex += 4
	}

	return &menuDto, nil
}
