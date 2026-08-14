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
	// TODO: menuDTO owner information and menu item names, description, allergens should be converted to a string
	// representation, translated by google translate api and then parsed back into the menu owner info, item names,
	// descriptions and allergens fields.
	// Direct replacement, not introducing new fields.

	menuStringRep := []string{}
	menuStringRep = append(menuStringRep, menuDto.MenuOwner.Name)
	menuStringRep = append(menuStringRep, menuDto.MenuOwner.Slogan)
	menuStringRep = append(menuStringRep, menuDto.MenuConfiguration.CategoryOrder)

	for _, item := range menuDto.MenuItems {
		menuStringRep = append(menuStringRep, item.Name)
		menuStringRep = append(menuStringRep, item.Description)
		menuStringRep = append(menuStringRep, item.Allergens)
		menuStringRep = append(menuStringRep, item.Category)
	}

	text := strings.Join(menuStringRep, "{0}")

	parsedLanguageTag, err := language.Parse(targetLanguage)
	if err != nil {
		return nil, err
	}

	translatedStringRep, err := s.translateClient.Translate(s.ctx, []string{text}, parsedLanguageTag, nil)
	if err != nil {
		return nil, err
	}

	fmt.Println("TRANSLATED:", translatedStringRep[0].Text)

	translatedStringParts := strings.Split(translatedStringRep[0].Text, "{0}")

	menuDto.MenuOwner.Name = translatedStringParts[0]
	menuDto.MenuOwner.Slogan = translatedStringParts[1]
	menuDto.MenuConfiguration.CategoryOrder = translatedStringParts[2]

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
