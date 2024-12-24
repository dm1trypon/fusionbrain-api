# go-fusionbrain-api

The `go-fusionbrain-api` package provides a Go client for interacting with the Fusion Brain API, which allows users to access artificial intelligence models for tasks such as text-to-image generation. This package abstracts the details of making HTTP calls to the API and allows users to focus on integrating the features into their applications.

## Table of Contents

- [Features](#features)
- [Installation](#installation)
- [Usage](#usage)
  - [Creating a Fusion Brain Client](#creating-a-fusion-brain-client)
  - [Checking Model Availability](#checking-model-availability)
  - [Getting Models](#getting-models)
  - [Getting Styles](#getting-styles)
  - [Text to Image Generation](#text-to-image-generation)
  - [Checking Generation Status](#checking-generation-status)

## Features

- Check service availability for models.
- Fetch available models.
- Fetch styles for image generation.
- Generate images from text descriptions.
- Check the status of image generation tasks.

## Installation

To install the `go-fusionbrain-api` package, you can use the following command:

```bash
go get github.com/dm1trypon/go-fusionbrain-api
```

## Usage

### Creating a Fusion Brain Client

To begin using the API, you'll need to create an instance of the `FusionBrain` client by providing the required API keys:

```go
package main

import (
	"context"
	"log"

	"github.com/yourusername/go-fusionbrain-api/fusionbrain"
	"github.com/valyala/fasthttp"
)

func main() {
	client := fasthttp.Client{}
	fusionClient := fusionbrain.NewFusionBrain(&client, "your-api-key", "your-secret-key")

	// Use the client for various functionalities.
}
```

### Checking Model Availability

You can check if the service is available to accept new tasks:

```go
availableErr := fusionClient.CheckAvailable(context.Background(), modelID)
if availableErr != nil {
	log.Printf("Service is unavailable: %v", availableErr)
} else {
	log.Println("Service is available.")
}
```

### Getting Models

Fetch a list of available models:

```go
models, err := fusionClient.GetModels(context.Background())
if err != nil {
	log.Fatalf("Error fetching models: %v", err)
}

for _, model := range models {
	log.Printf("Model: %s, Version: %s", model.Name, model.Version)
}
```

### Getting Styles

Get the current list of styles available for image generation:

```go
styles, err := fusionClient.GetStyles(context.Background())
if err != nil {
	log.Fatalf("Error fetching styles: %v", err)
}

for _, style := range styles {
	log.Printf("Style: %s, Title: %s", style.Name, style.Title)
}
```

### Text to Image Generation

Generate an image based on a text description:

```go
requestBody := fusionbrain.RequestBody{
	Prompt:        "A beautiful sunset over the mountains",
	NegativePrompt: "No people",
	Style:        "fantasy",
	Width:        512,
	Height:       512,
}

uuid, err := fusionClient.TextToImage(context.Background(), requestBody, modelID)
if err != nil {
	log.Fatalf("Error generating image: %v", err)
}
log.Printf("Image generation requested with UUID: %s", uuid)
```

### Checking Generation Status

Check the status of an image generation task using its UUID:

```go
status, err := fusionClient.CheckStatus(context.Background(), uuid)
if err != nil {
	log.Fatalf("Error checking status: %v", err)
}

log.Printf("Generation Status: %s", status.Status)
```