# Golang Plugin Boilerplate

## Overview

This repository is a boilerplate for creating Go-based projects with Kubernetes. It provides a structure to help you start quickly with Go, Kubernetes, and microservices development. The boilerplate includes basic CRUD endpoints and Swagger documentation.
## Quick Start

1. **Clone the Repository:**
    ```bash
    git clone https://github.com/LerianStudio/golang-plugin-boilerplate.git
    cd goland-plugin-boilerplate
    ```

2. **Setup environment variables:**
    ```bash
    make set-env
    ```

3. **Run the Server:**
    ```bash
    make up
    ```

4. **Access the API:**
   Visit `http://localhost:4000` to interact with the API.

## Swagger Documentation

The boilerplate includes Swagger documentation that helps in visualizing and interacting with the API endpoints. You can access the documentation by running the project and navigating to `http://localhost:4000/swagger/index.html`.

The API documentation provides detailed information about:
- Available endpoints
- Request parameters and body schemas
- Response formats and status codes
- Data models and definitions

This documentation is automatically generated from the API code and is always up-to-date with the current implementation.