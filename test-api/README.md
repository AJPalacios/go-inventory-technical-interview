# 📡 API Testing Guide

Simple HTTP test files for the Inventory API.

## 🚀 Quick Start

1. **Install VSCode REST Client Extension**
   - Search "REST Client" in VSCode Extensions

2. **Start Server**
   ```bash
   make run  # Server starts on http://localhost:8080
   ```

3. **Open any `.http` file and click "Send Request"**

## 📁 Test Files

| File | Purpose | What It Tests |
|------|---------|---------------|
| **`inventory-api.http`** | Basic API operations | Health, stock checks, reserve/release, errors |
| **`complete-workflows.http`** | Business scenarios | E-commerce order, cart abandonment, restocking |
| **`validation-errors.http`** | Error handling | Invalid inputs, missing fields, format errors |

## 🎯 Test Products

```
e08e3e7e-9126-49e4-9caf-63885a07bd78  # Teclado Keychron K2 (main test product)
2d70d1dc-cd3a-4f40-afb0-52e16445bf36  # Laptop HP Pavilion 15
2da3b8d3-69f1-46e6-a068-874532d5126a  # Mouse Logitech MX Master 3
```

## 💡 Tips

- Start with **health check** to verify server is running
- **Copy reservation_id** from reserve responses for release tests  
- Check stock levels **before and after** operations
- Use realistic quantities (1-10)
- Interactive docs at: **http://localhost:8080/docs**

## 🔄 Business Workflows

**A. E-commerce Success**: Check stock → Reserve → Purchase → Complete  
**B. Cart Abandonment**: Reserve → Customer leaves → Cancel → Release  
**C. Warehouse Restock**: Check inventory → Receive shipment → Update stock

---

**📖 Full API documentation: http://localhost:8080/docs\*\*
