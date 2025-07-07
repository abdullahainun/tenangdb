# TenangDB Website

Modern, clean website for TenangDB - built for GitHub Pages deployment.

## 🚀 Live Website

Visit the live website at: https://abdullahainun.github.io/tenangdb

## 🏗️ Architecture

- **Static Site**: Pure HTML, CSS, and JavaScript
- **Modern Design**: Clean, responsive, and accessible
- **Performance Optimized**: Fast loading with minimal dependencies
- **GitHub Pages Ready**: Configured for automatic deployment

## 📁 Structure

```
docs/website/
├── index.html              # Main landing page
├── assets/
│   ├── css/
│   │   └── style.css       # Modern CSS styling
│   ├── js/
│   │   └── main.js         # Interactive JavaScript
│   └── images/
│       └── favicon.svg     # Brand favicon
├── _config.yml             # GitHub Pages configuration
└── README.md               # This file
```

## 🎨 Features

### Design
- **Modern Gradient Design**: Clean, professional appearance
- **Responsive Layout**: Works on all devices
- **Interactive Elements**: Smooth animations and transitions
- **Terminal Demo**: Animated code examples
- **Dark/Light Theme**: Optimized for readability

### Content Sections
- **Hero Section**: Value proposition and quick start
- **Features Grid**: Key benefits and capabilities  
- **Installation Guide**: Step-by-step setup instructions
- **Documentation Links**: Direct links to all docs
- **GitHub Integration**: Real-time stars counter

### Technical
- **SEO Optimized**: Meta tags, Open Graph, structured data
- **Performance**: Optimized images, minimal dependencies
- **Accessibility**: WCAG compliant, keyboard navigation
- **Analytics Ready**: Google Analytics integration

## 🚀 GitHub Pages Deployment

### Automatic Deployment
1. Push changes to `main` branch
2. GitHub Pages automatically builds and deploys
3. Website updates in 1-2 minutes

### Manual Setup
1. Go to repository Settings
2. Navigate to Pages section
3. Source: Deploy from a branch
4. Branch: `main`
5. Folder: `/docs/website`
6. Save settings

### Custom Domain (Optional)
1. Add `CNAME` file with your domain
2. Configure DNS records
3. Enable HTTPS in GitHub Pages settings

## 🛠️ Local Development

### Prerequisites
- Web server (Python, Node.js, or any HTTP server)
- Modern web browser

### Run Locally
```bash
# Using Python
cd docs/website
python -m http.server 8000

# Using Node.js
npx serve .

# Using PHP
php -S localhost:8000
```

Then visit: http://localhost:8000

## 📝 Content Updates

### Updating Content
- Edit `index.html` for main content
- Modify `assets/css/style.css` for styling
- Update `assets/js/main.js` for functionality

### Adding New Sections
1. Add HTML section in `index.html`
2. Add corresponding styles in `style.css`
3. Add navigation link if needed

### Documentation Links
All documentation links point to GitHub repository files:
- Installation: `/INSTALL.md`
- Configuration: `/configs/README.md`
- Commands: `/docs/COMMANDS.md`
- Security: `/docs/SECURITY.md`

## 🎯 SEO & Analytics

### SEO Features
- Semantic HTML structure
- Meta descriptions and keywords
- Open Graph tags for social sharing
- Structured data for search engines

### Analytics Setup
1. Get Google Analytics tracking ID
2. Add to `_config.yml` or directly in HTML
3. Monitor traffic and user behavior

## 🔧 Customization

### Colors & Branding
Update CSS variables in `style.css`:
```css
:root {
  --primary-color: #667eea;
  --secondary-color: #764ba2;
  --accent-color: #f093fb;
}
```

### Content Sections
- **Hero**: Update value proposition and taglines
- **Features**: Modify feature cards and descriptions
- **Installation**: Update installation steps
- **Footer**: Add additional links or information

### Interactive Elements
- **Terminal**: Modify demo commands in HTML
- **Animations**: Adjust CSS animations and transitions
- **JavaScript**: Add new interactive features

## 🚀 Performance Optimization

### Current Optimizations
- **Minimal Dependencies**: Only Google Fonts
- **Optimized Images**: SVG favicon, minimal graphics
- **CSS Grid**: Modern layout techniques
- **Lazy Loading**: Images load as needed

### Additional Optimizations
- **CDN**: Consider using a CDN for assets
- **Compression**: Enable gzip compression
- **Caching**: Set appropriate cache headers
- **Minification**: Minify CSS and JavaScript

## 📊 Analytics & Monitoring

### Metrics to Track
- **Page Views**: Overall traffic
- **Bounce Rate**: User engagement
- **Download Clicks**: Installation attempts
- **GitHub Clicks**: Repository visits

### Conversion Goals
- GitHub repository visits
- Documentation page views
- Installation guide completion
- Issue reports or contributions

## 🤝 Contributing

### Making Changes
1. Fork the repository
2. Create feature branch
3. Make changes to website files
4. Test locally
5. Submit pull request

### Content Guidelines
- Keep messaging clear and concise
- Maintain brand consistency
- Ensure mobile responsiveness
- Test across different browsers

## 📄 License

Website content is licensed under MIT License - same as TenangDB project.