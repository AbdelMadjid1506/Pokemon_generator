package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Pokemon struct {
	Name        string         `json:"name"`
	Types       []string       `json:"types"`
	Description string         `json:"description"`
	Abilities   []string       `json:"abilities"`
	Stats       map[string]int `json:"stats"`
}

type OllamaChunk struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

func main() {
	prompt := `Génère un Pokémon original au format JSON strict :
Le Pokémon doit avoir un nom, des types, une description en FRANÇAIS,
une liste de capacités, et des statistiques numériques.
{
  "name": "string",
  "types": ["string", "string"],
  "description": "string",
  "abilities": ["string", "string"],
  "stats": {"hp": 0, "attack": 0, "defense": 0, "speed": 0}
}`

	err := godotenv.Load()
	if err != nil {
		log.Fatal("⚠️ Erreur chargement .env")
	}
	apiKey := os.Getenv("HF_API_KEY")
	if apiKey == "" {
		log.Fatal("⚠️ HF_API_KEY manquant (exporte ton token HuggingFace)")
	}

	reqBody := map[string]interface{}{
		"model":  "phi3",
		"prompt": prompt,
	}
	body, _ := json.Marshal(reqBody)

	resp, err := http.Post("http://localhost:11434/api/generate", "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Fatal("Erreur requête Ollama:", err)
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	var fullResponse string
	for scanner.Scan() {
		var chunk OllamaChunk
		if err := json.Unmarshal(scanner.Bytes(), &chunk); err != nil {
			continue
		}
		fullResponse += chunk.Response
		if chunk.Done {
			break
		}
	}

	// Nettoyer les ``` éventuels
	fullResponse = strings.TrimPrefix(fullResponse, "```json")
	fullResponse = strings.TrimPrefix(fullResponse, "```")
	fullResponse = strings.TrimSuffix(fullResponse, "```")

	fmt.Println("Réponse combinée Ollama:\n", fullResponse)

	// Parser en Pokémon
	var p Pokemon
	if err := json.Unmarshal([]byte(fullResponse), &p); err != nil {
		log.Printf("⚠️ JSON invalide: %v\nTexte reçu:\n%s", err, fullResponse)
	} else {
		fmt.Printf("\n✅ Pokémon généré: %+v\n", p)
	}

	description := p.Description

	url := "https://api-inference.huggingface.co/models/stabilityai/stable-diffusion-xl-base-1.0"

	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(fmt.Sprintf(`{"inputs": "%s"}`, description))))
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	// Envoyer la requête
	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()

	// Sauvegarder l’image en sortie
	img, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Code HTTP:", resp.StatusCode)
	bodyPreview := string(img[:200]) // affiche les 200 premiers caractères
	fmt.Println("Réponse HuggingFace:", bodyPreview)
	err = os.WriteFile("pokemon.png", img, 0644)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("✅ Image générée et sauvegardée dans pokemon.png")

}
