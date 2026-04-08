# ChatAPI Documentation

This directory contains the Hugo-based documentation site for ChatAPI.

## Structure

```
docs/
├── content/           # Documentation pages
│   ├── _index.md     # Homepage
│   ├── getting-started/
│   ├── api/          # API documentation
│   ├── guides/       # User guides
│   └── architecture/ # System architecture
├── static/            # Static assets
│   └── api/
│       └── openapi.yaml  # OpenAPI specification
├── themes/            # Hugo themes (git submodules)
├── hugo.toml          # Hugo configuration
└── config/            # Additional config (if needed)
```

## Local Development

### Prerequisites

- [Hugo](https://gohugo.io/getting-started/installing/) (extended version)
- Git

### Running the Documentation Site

```bash
# Navigate to docs directory
cd docs

# Start development server
hugo server

# Or with drafts enabled
hugo server -D
```

The site will be available at `http://localhost:1313`

### Building for Production

```bash
# Build static files
hugo

# Build with minification
hugo --minify
```

Static files will be generated in the `public/` directory.

## Writing Documentation

### Page Structure

All documentation pages use Hugo's content organization:

- `_index.md` files create section index pages
- Regular `.md` files create individual pages
- Frontmatter uses TOML format

### Frontmatter Example

```toml
+++
title = "Page Title"
weight = 10
draft = false
+++
```

### Shortcodes

The site uses the [Hugo Book theme](https://github.com/alex-shpak/hugo-book) which provides several shortcodes. However, some shortcodes may not be available in the current setup.

### Adding New Pages

```bash
# Create a new page
hugo new section/page.md

# Create a new section
hugo new section/_index.md
```

## API Documentation

### Updating API Docs

1. Update the documentation pages in `content/api/`

## Deployment

### GitHub Pages

Documentation is automatically deployed to GitHub Pages via GitHub Actions when changes are pushed to the `main` branch.

The workflow file is located at `.github/workflows/deploy-docs.yml`.

### Manual Deployment

```bash
# Build the site
cd docs
hugo

# Deploy the public/ directory to your hosting provider
```

## Theme

The site uses the [Hugo Book theme](https://github.com/alex-shpak/hugo-book), installed as a git submodule.

### Updating the Theme

```bash
# Update the theme submodule
git submodule update --remote themes/hugo-book

# Check for breaking changes in the theme documentation
```

## Contributing

### Guidelines

- Use clear, concise language
- Include code examples where helpful
- Keep pages focused on single topics
- Test all links and examples
- Use consistent formatting

### Code Examples

- Include syntax highlighting for code blocks
- Test all code examples
- Use realistic, working examples
- Include comments for complex examples

### Images and Assets

- Place images in the `static/` directory
- Use descriptive filenames
- Optimize images for web
- Include alt text for accessibility

## Troubleshooting

### Common Issues

**Shortcode errors**: Some Hugo Book shortcodes may not be available. Check the theme documentation or use standard Markdown.

**Build failures**: Ensure all frontmatter is valid TOML and all links are correct.

**Theme issues**: Update the theme submodule or check for compatibility.

### Getting Help

- [Hugo Documentation](https://gohugo.io/documentation/)
- [Hugo Book Theme](https://github.com/alex-shpak/hugo-book)
- [OpenAPI Specification](https://swagger.io/specification/)

## Maintenance

### Regular Tasks

- Update Hugo version periodically
- Review and update theme submodule
- Check for broken links
- Update API documentation when endpoints change
- Refresh code examples

### Versioning

Documentation versions correspond to ChatAPI releases. Create branches for major version documentation if needed.