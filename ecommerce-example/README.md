# E-commerce Example (kpt)

This is a simple e-commerce application example packaged using **kpt**.

---

## 📌 Overview

This project demonstrates how Kubernetes applications can be managed and customized using the **kpt toolchain**.

Instead of modifying code, users can change configuration using YAML files.

---

## 🚀 Features

- Simple Kubernetes Deployment using **nginx** (version pinned for reproducibility)  
- Service to expose the application internally  
- Easy-to-understand structure for beginners  
- Ready to extend with custom configuration

---

## 📂 Structure

- `deployment.yaml` → Defines the application Deployment  
- `service.yaml` → Exposes the application via a Service  
- `Kptfile` → Defines the kpt package metadata  

---

## ⚙️ How to Use

### 1. Deploy the application

**Option A: Using kubectl only**
```bash
kubectl apply -f deployment.yaml -f service.yaml
