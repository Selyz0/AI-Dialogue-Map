# AI Dialogue Map (Development Version)

AI Chat Dialogue Tree Application (Development version)

## Overview

AI Dialogue Map is a desktop application that utilizes Google's Gemini API to conduct dialogues with AI and visually record and manage the flow of these conversations as a tree structure. It displays user questions and AI responses as nodes, making it easy to track dialogue branches and delve deeper into topics. The generated dialogue trees can be saved and loaded on a per-project basis.

## Setup and Installation

### Prerequisites

* Go (version 1.18 or higher recommended)
* Fyne library (version 2.4.0 or higher recommended)

### Installation Steps

1.  **Set Up API Key:**
    * Create a file named `secret.toml` under `{root}/internal/config/` directory of this application.
    * Add the following content to `secret.toml`, replacing `YOUR_GEMINI_API_KEY` with your actual Gemini API key:
        ```toml
        gemini_api_key = "YOUR_GEMINI_API_KEY"
        ```
    * You can obtain an API key from Google AI Studio ([https://aistudio.google.com/](https://aistudio.google.com/)) or other sources.

## File Structure (Source Code)

The application is divided into the following files by functionality (all within the `main` package):

* `main.go`: Main
* `service.go`: Service logic, UI assembly.
* `config.go`: Configuration file loading.
* `ai_client.go`: Gemini API client.
* `theme.go`: Custom theme definition.
* `node_widget.go`: Node data structure (`NodeData`) and UI widget (`NodeWidget`).
* `dialog_canvas.go`: Custom canvas (`DialogCanvas`) for displaying the dialogue tree.
* `utils.go`: Utility functions.

## Usage

1.  **Start the Application:** Launch the application according to the "How to Run" section.
2.  **Start a New Dialogue:**
    * Enter your first question to the AI in the text input area at the bottom of the screen.
    * Click the "Send" button or press Ctrl+Enter. An AI response will be generated, and the first node will be created. A new project will also be automatically created, with its name derived from the AI's response.
3.  **Continue and Branch Dialogues:**
    * Click on an existing node to select it. It will be highlighted and set as the source for new branches.
    * Submitting a new question while a node is selected will create a new node branching from the selected one.
    * Submitting a question without a node selected may create a new independent tree or connect to the root of the last interacted tree (current implementation primarily connects to the selected branch source).
4.  **Node Operations:**
    * **Expand/Collapse:** Click the vertical three-dot icon (or downward arrow when expanded) in the bottom-right of each node to expand or collapse the display of the answer content.
    * **Drag & Drop:** Drag nodes with the mouse to freely change their position on the canvas.
    * **Create Branch:** Click the "+" icon on the right side of a node to select it as the branch source.
    * **Delete:** Click the trash can icon in the top-right of a node. After a confirmation dialog, the node and all its descendants will be deleted.
5.  **Canvas Operations:**
    * **Pan:** Hold the Ctrl key and drag the canvas background to move the viewable area up, down, left, or right.
    * **Zoom:** Hold the Ctrl key and scroll the mouse wheel up or down to zoom the entire canvas in or out.
6.  **Saving Projects:**
    * The current project is automatically saved when new nodes are added or existing nodes are deleted.
    * You can also manually save the current project by selecting "File" -> "Save Project" from the menu bar.
7.  **Loading Projects:**
    * Select "File" -> "Open Project..." from the menu bar.
    * Choose a previously saved project from the displayed dialog to open it.
8.  **Creating a New Project (Manual):**
    * Select "File" -> "New Project" from the menu bar. This will clear the current workspace, allowing you to start a new project.

## Future Enhancements (Partial List)

* Editing functionality for existing node titles, questions, and AI answers.
* Project duplication, export/import features.
* Implementation of more advanced node auto-layout algorithms.
* Search functionality (for node content, titles, etc.).

## Known Issues/Limitations

* Performance may be affected when dealing with a large number of nodes with very long text content.
* The behavior of Japanese IME input may vary slightly depending on the OS and Fyne version.

---

We hope this application helps you in your idea generation and knowledge organization.