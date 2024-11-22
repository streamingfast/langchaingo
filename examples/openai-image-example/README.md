# OpenAI Vision Analysis Example with LangChain Go

Welcome to this example of using OpenAI's GPT-4 Vision model with LangChain Go! 🎉

This project demonstrates how to analyze images using OpenAI's vision capabilities through LangChain Go's integration.

## What This Example Does

This example showcases several key features:

1. 🖼️ Loads and processes local image files
2. 🔄 Converts images to base64 format for API compatibility
3. 🤖 Connects to OpenAI's GPT-4 Vision model
4. 📝 Creates a chain for image analysis with custom prompting
5. 🎯 Outputs AI-generated image descriptions

## How It Works

1. Creates an OpenAI client with the GPT-4 Vision model
2. Processes images by:
   - Reading local PNG files
   - Converting to base64 format
   - Preparing for API submission
3. Creates an analysis chain with:
   - System prompt defining the AI as an image analysis expert
   - Human prompt for image description requests
   - Image placeholder for dynamic image injection

## Running the Example

To run this example, you'll need:

1. Go installed on your system
2. Environment variables set up:
   - `OPENAI_API_KEY` - Your OpenAI API key

```bash
go run .
```
