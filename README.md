<h1 align="center">
  <a href="https://safebucket.io"><img src="./assets/safebucket_banner.png" alt="SafeBucket"></a>
</h1>

## 📖 Introduction

SafeBucket is an open-source secure file sharing platform designed to share files in an easy and secure way, integrating with different cloud providers. Built for individuals and organizations that need to collaborate on files with robust security, flexible access controls, and seamless multi-cloud support across AWS S3, Google Cloud Storage, and MinIO.

![SafeBucket Homepage](./assets/homepage.png)

## ✨ Features

- 🔒 **Secure File Sharing**: Share files and folders with colleagues, clients, and teams through secure bucket-based collaboration
- ☁️ **Multi-Storage Integration**: Store and share files across AWS S3, GCP Cloud Storage, or MinIO
- 🔐 **OIDC Integration**: Single sign-on with any OIDC provider for seamless authentication
- 👥 **Role-Based Access Control**: Granular sharing permissions with owner, contributor, and viewer roles
- 📧 **User Invitation System**: Invite collaborators via email with secure role-based access to shared buckets
- 📊 **Real-Time Activity Tracking**: Monitor file sharing activity with comprehensive audit trails
- 🚀 **Highly Scalable**: Event-driven architecture for high-performance operations

SafeBucket simplifies secure file sharing by providing an intuitive platform that combines the convenience of modern collaboration tools with the security and flexibility of self-hosted or cloud storage solutions.

## 🚀 Quick Start

### Prerequisites
- Go 1.23+
- Node.js 22+
- Docker & Docker Compose (optional)

### Installation

1. **Clone the repository**
   ```bash
   git clone https://github.com/safebucket/safebucket.git
   cd safebucket
   ```

2. **Backend Setup**
   ```bash
   # Install dependencies and run
   go mod tidy
   go run main.go
   ```

3. **Frontend Setup**
   ```bash
   cd web
   npm install
   npm run dev
   ```

4. **Using Docker Compose**
   ```bash
   docker-compose up -d
   ```

## 🔧 Development

### Backend Commands
```bash
go run main.go                 # Start development server
go test ./...                  # Run tests
go fmt ./...                   # Format code
go mod tidy                    # Clean dependencies
```

### Frontend Commands
```bash
npm run dev                    # Development server with HMR
npm run build                  # Production build
npm run fixup                  # Prettier fix
```

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guidelines](CONTRIBUTING.md) for details.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Built with ❤️ using Go and React
- Icons by [Lucide](https://lucide.dev)
- UI components by [Radix UI](https://radix-ui.com) and [shadcn/ui](https://ui.shadcn.com)

## 📞 Support

- 📧 Email: support@safebucket.io
- 🐛 Issues: [GitHub Issues](https://github.com/yourusername/safebucket/issues)
