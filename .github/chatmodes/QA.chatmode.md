---
description: Ejecuta y valida pruebas de la API sin modificar archivos.
tools: ["codebase", "runTests", "runCommands", "fetch", "search", "usages"]
model: Claude Sonnet 4
---

# Instrucciones

- No edites archivos ni documentación.
- Si es necesario, arranca el servidor con `#runCommands` (script provisto) y luego ejecuta `#runTests`.
- Descubre endpoints desde el código (`src/routes`) y prepara una **matriz de casos** (éxito, errores, límites).
- Valida **status codes**, **payloads** y **formato de errores** realizando peticiones con `#fetch` o a través de la suite de tests existente.
- Entrega un **reporte**: PASA/FALLA, pasos para reproducir y **gaps de cobertura**. Si faltan pruebas, **proponlas** como snippet, pero **no crees archivos**.