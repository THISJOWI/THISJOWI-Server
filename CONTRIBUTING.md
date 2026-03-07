# 🤝 Contributing to THISJOWI Backend

First off, thank you for considering contributing to THISJOWI Backend! It's people like you that make this project better for everyone.

## 📋 Table of Contents

- [Code of Conduct](#-code-of-conduct)
- [Getting Started](#-getting-started)
- [How Can I Contribute?](#-how-can-i-contribute)
- [Development Workflow](#-development-workflow)
- [Style Guidelines](#-style-guidelines)
- [Commit Messages](#-commit-messages)
- [Pull Request Process](#-pull-request-process)
- [Community](#-community)

## 📜 Code of Conduct

### Our Pledge

We pledge to make participation in our project a harassment-free experience for everyone, regardless of age, body size, disability, ethnicity, gender identity and expression, level of experience, nationality, personal appearance, race, religion, or sexual identity and orientation.

### Our Standards

✅ **Positive Behavior:**
- Using welcoming and inclusive language
- Being respectful of differing viewpoints
- Gracefully accepting constructive criticism
- Focusing on what is best for the community
- Showing empathy towards other community members

❌ **Unacceptable Behavior:**
- Trolling, insulting/derogatory comments, and personal or political attacks
- Public or private harassment
- Publishing others' private information without permission
- Other conduct which could reasonably be considered inappropriate

## 🚀 Getting Started

### Prerequisites

Before you begin, ensure you have:

- [ ] Java 17 or higher installed
- [ ] Maven 3.8+ installed
- [ ] Git configured with your name and email
- [ ] A GitHub account
- [ ] Basic understanding of Spring Boot and microservices

### Fork and Clone

```bash
# Fork the repository on GitHub
# Then clone your fork
git clone https://github.com/YOUR_USERNAME/THISJOWI-backend.git
cd THISJOWI-backend

# Add upstream remote
git remote add upstream https://github.com/ORIGINAL_OWNER/THISJOWI-backend.git
```

### Setup Development Environment

```bash
# Build all services
./mvnw clean install

# Run tests
./mvnw test

# Start Eureka (required for development)
cd Eureka
./mvnw spring-boot:run
```

## 🎯 How Can I Contribute?

### 🐛 Reporting Bugs

Before creating bug reports, please check the [issue list](../../issues) to avoid duplicates.

**Great Bug Report Includes:**
- Clear, descriptive title
- Exact steps to reproduce the problem
- Expected vs actual behavior
- Screenshots (if applicable)
- Environment details (OS, Java version, etc.)

**Example:**

```markdown
**Bug:** Authentication service returns 500 on valid login

**Steps to Reproduce:**
1. Start auth service
2. Send POST to /api/auth/login with valid credentials
3. Observe 500 error response

**Expected:** 200 OK with JWT token
**Actual:** 500 Internal Server Error

**Environment:**
- OS: Ubuntu 22.04
- Java: 17.0.8
- Spring Boot: 3.1.5

**Logs:**
```
[Paste relevant logs here]
```
```

### 💡 Suggesting Enhancements

Enhancement suggestions are tracked as GitHub issues.

**Include:**
- Use a clear, descriptive title
- Detailed description of the proposed feature
- Use cases and examples
- Why this enhancement would be useful
- Possible implementation approach

### 📝 Your First Code Contribution

Unsure where to begin? Look for issues labeled:
- `good first issue` - Simple issues for newcomers
- `help wanted` - Issues we need help with
- `documentation` - Improve or create documentation

### 🔧 Pull Requests

We actively welcome your pull requests!

**Areas for Contribution:**
- Bug fixes
- New features
- Performance improvements
- Documentation improvements
- Test coverage
- Code refactoring

## 🔄 Development Workflow

### 1. Create a Feature Branch

```bash
# Update your local main
git checkout main
git pull upstream main

# Create a new branch
git checkout -b feature/your-feature-name
# or
git checkout -b fix/bug-description
# or
git checkout -b docs/what-you-are-documenting
```

**Branch Naming Convention:**
- `feature/` - New features
- `fix/` - Bug fixes
- `docs/` - Documentation changes
- `refactor/` - Code refactoring
- `test/` - Adding or updating tests
- `chore/` - Maintenance tasks

### 2. Make Your Changes

```bash
# Make your changes
# ...

# Run tests
./mvnw test

# Check code style
./mvnw checkstyle:check

# Build to ensure everything compiles
./mvnw clean package
```

### 3. Commit Your Changes

```bash
# Stage your changes
git add .

# Commit with a descriptive message
git commit -m "feat: add user profile endpoint"
```

See [Commit Messages](#-commit-messages) for guidelines.

### 4. Push and Create PR

```bash
# Push to your fork
git push origin feature/your-feature-name

# Go to GitHub and create a Pull Request
```

## 🎨 Style Guidelines

### Java Code Style

We follow the [Google Java Style Guide](https://google.github.io/styleguide/javaguide.html) with minor modifications.

**Key Points:**
- **Indentation:** 4 spaces (no tabs)
- **Line Length:** 120 characters max
- **Braces:** K&R style (opening brace on same line)
- **Naming:**
  - Classes: `PascalCase`
  - Methods/Variables: `camelCase`
  - Constants: `UPPER_SNAKE_CASE`
  - Packages: `lowercase`

**Example:**

```java
@Service
public class UserService {
    private static final int MAX_RETRY_ATTEMPTS = 3;
    
    private final UserRepository userRepository;
    
    public UserService(UserRepository userRepository) {
        this.userRepository = userRepository;
    }
    
    public User findUserById(Long userId) {
        return userRepository.findById(userId)
            .orElseThrow(() -> new UserNotFoundException(userId));
    }
}
```

### Spring Boot Best Practices

- ✅ Use constructor injection (not `@Autowired` on fields)
- ✅ Keep controllers thin, logic in services
- ✅ Use DTOs for API requests/responses
- ✅ Validate input with `@Valid`
- ✅ Handle exceptions with `@ControllerAdvice`
- ✅ Use proper HTTP status codes
- ✅ Document APIs with Swagger/OpenAPI

### File Organization

```
src/
├── main/
│   ├── java/
│   │   └── uk/thisjowi/ServiceName/
│   │       ├── config/          # Configuration classes
│   │       ├── controller/      # REST controllers
│   │       ├── dto/             # Data Transfer Objects
│   │       ├── entity/          # JPA entities
│   │       ├── repository/      # Data access layer
│   │       ├── service/         # Business logic
│   │       ├── exception/       # Custom exceptions
│   │       └── utils/           # Utility classes
│   └── resources/
│       ├── application.yaml     # Configuration
│       └── ...
└── test/
    └── java/
        └── uk/thisjowi/ServiceName/
            ├── controller/      # Controller tests
            ├── service/         # Service tests
            └── integration/     # Integration tests
```

### Testing Standards

- ✅ Write unit tests for all business logic
- ✅ Integration tests for critical flows
- ✅ Aim for 80%+ code coverage
- ✅ Use meaningful test names: `should_ReturnUser_When_ValidIdProvided()`
- ✅ Follow AAA pattern: Arrange, Act, Assert

**Example:**

```java
@Test
void should_CreateUser_When_ValidDataProvided() {
    // Arrange
    UserDTO userDTO = new UserDTO("test@example.com", "password123");
    
    // Act
    User result = userService.createUser(userDTO);
    
    // Assert
    assertNotNull(result);
    assertEquals("test@example.com", result.getEmail());
    verify(userRepository, times(1)).save(any(User.class));
}
```

## 💬 Commit Messages

We follow [Conventional Commits](https://www.conventionalcommits.org/).

### Format

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Types

- `feat:` - New feature
- `fix:` - Bug fix
- `docs:` - Documentation changes
- `style:` - Code style changes (formatting, missing semi colons, etc)
- `refactor:` - Code refactoring
- `test:` - Adding or updating tests
- `chore:` - Maintenance tasks
- `perf:` - Performance improvements
- `ci:` - CI/CD changes

### Examples

```bash
# Simple feature
git commit -m "feat: add user profile endpoint"

# Bug fix with scope
git commit -m "fix(auth): resolve JWT token expiration issue"

# Breaking change
git commit -m "feat!: change authentication response structure

BREAKING CHANGE: The auth response now returns user object nested under 'data' key"

# Detailed commit
git commit -m "feat(notes): add note sharing functionality

- Add share endpoint POST /v1/notes/{id}/share
- Implement permission checks
- Add unit and integration tests
- Update API documentation

Closes #123"
```

## 🔍 Pull Request Process

### Before Submitting

- [ ] Code follows the style guidelines
- [ ] Self-review of your own code
- [ ] Commented code, particularly in hard-to-understand areas
- [ ] Updated documentation (if needed)
- [ ] Added tests that prove your fix/feature works
- [ ] All tests pass locally (`./mvnw test`)
- [ ] No new warnings or errors
- [ ] PR description clearly describes the changes

### PR Template

When creating a PR, please include:

```markdown
## Description
Brief description of what this PR does

## Type of Change
- [ ] Bug fix (non-breaking change which fixes an issue)
- [ ] New feature (non-breaking change which adds functionality)
- [ ] Breaking change (fix or feature that would cause existing functionality to not work as expected)
- [ ] Documentation update

## How Has This Been Tested?
Describe the tests you ran

## Checklist
- [ ] My code follows the style guidelines
- [ ] I have performed a self-review
- [ ] I have commented my code, particularly in hard-to-understand areas
- [ ] I have made corresponding changes to the documentation
- [ ] My changes generate no new warnings
- [ ] I have added tests that prove my fix is effective or that my feature works
- [ ] New and existing unit tests pass locally with my changes

## Screenshots (if applicable)

## Related Issues
Closes #123
Related to #456
```

### Review Process

1. **Automated Checks** - CI/CD pipeline runs tests
2. **Code Review** - At least one maintainer reviews
3. **Discussion** - Address review comments
4. **Approval** - Maintainer approves PR
5. **Merge** - PR is merged to main

**Review Timeline:**
- We aim to review PRs within **2-3 business days**
- Complex PRs may take longer
- Feel free to ping if no response after a week

## 👥 Community

### Communication Channels

- **GitHub Issues** - Bug reports and feature requests
- **Pull Requests** - Code contributions and discussions
- **Discussions** - General questions and ideas

### Getting Help

**Need help? Try these steps:**

1. **Check Documentation**
   - Read this CONTRIBUTING guide
   - Review the [README.md](README.md)
   - Check [SECURITY.md](SECURITY.md)

2. **Search Existing Issues**
   - Someone might have had the same question
   - Look for `question` label

3. **Ask in Discussions**
   - Start a new discussion
   - Be clear and provide context

4. **Visit Our Website**
   - Comprehensive tutorials and guides
   - Architecture documentation
   - Best practices
   
   **👉 [thisjowi.uk](https://thisjowi.uk)**

5. **Contact Maintainers**
   - For complex issues or private matters
   - Email: dev@thisjowi.uk

### Recognition

Contributors are recognized in:
- Contributors section of README
- Release notes
- GitHub contributors page

**Top contributors may be invited to become maintainers!**

## 📚 Additional Resources

### Learn More

- [Spring Boot Documentation](https://docs.spring.io/spring-boot/docs/current/reference/html/)
- [Spring Cloud Documentation](https://spring.io/projects/spring-cloud)
- [Microservices Patterns](https://microservices.io/patterns/index.html)
- [Kubernetes Documentation](https://kubernetes.io/docs/home/)

### External Guides

- [How to Write a Git Commit Message](https://chris.beams.io/posts/git-commit/)
- [GitHub Flow](https://guides.github.com/introduction/flow/)
- [Semantic Versioning](https://semver.org/)

### Tools We Use

- **IDE:** IntelliJ IDEA / Eclipse / VS Code
- **Build Tool:** Maven
- **Version Control:** Git
- **CI/CD:** GitHub Actions
- **Container:** Docker
- **Orchestration:** Kubernetes

## 🎓 Detailed Information

For comprehensive guides, tutorials, and in-depth documentation:

### 🌐 Visit Our Website

**[thisjowi.uk](https://thisjowi.uk)**

You'll find:
- 📖 Complete API documentation
- 🎥 Video tutorials
- 🏗 Architecture deep-dives
- 🔐 Security best practices
- 💡 Implementation examples
- 🚀 Deployment guides
- 📰 Latest updates and blog posts

---

## 🙏 Thank You!

Your contributions make this project better for everyone. Whether it's a bug report, feature request, documentation improvement, or code contribution - we appreciate your time and effort!

**Happy Coding! 🚀**

---

<div align="center">

**Questions? Need More Info?**

[Visit Our Website](https://thisjowi.uk) • [Read the Docs](README.md) • [Report Issues](../../issues)

Made with ❤️ by the THISJOWI Community

</div>
