E-commerce Example (kpt)

This is a simple e-commerce application example packaged using kpt.

📌 Overview

This project demonstrates how Kubernetes applications can be managed and customized using the kpt toolchain.

Instead of modifying code, users can change configuration using YAML files.

---

🚀 Features

- Simple Kubernetes deployment using nginx
- Service to expose the application
- Easy to understand structure for beginners
- Ready to extend with customization

---

📂 Structure

- "deployment.yaml" → Defines the application deployment
- "service.yaml" → Exposes the application
- "Kptfile" → Defines the kpt package

---

⚙️ How to Use

1. Deploy the application

kubectl apply -f .

---

2. Check status

kubectl get pods
kubectl get services

---

🎯 Goal

The goal of this example is to:

- Help beginners understand kpt
- Demonstrate “configuration as data”
- Provide a base for more advanced customization

---

🔮 Future Improvements

- Add configurable values (shop name, currency)
- Add multiple deployment sizes (small/medium/large)
- Add localization support

---

📖 Conclusion

This example is a starting point for learning how to use kpt to manage Kubernetes applications in a simple and structured way.
