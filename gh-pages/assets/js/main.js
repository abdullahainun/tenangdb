// Navigation Toggle
document.addEventListener('DOMContentLoaded', function() {
    const navToggle = document.querySelector('.nav-toggle');
    const navLinks = document.querySelector('.nav-links');
    
    if (navToggle && navLinks) {
        navToggle.addEventListener('click', function() {
            navLinks.classList.toggle('active');
        });
    }
    
    // Smooth scrolling for anchor links
    document.querySelectorAll('a[href^="#"]').forEach(anchor => {
        anchor.addEventListener('click', function (e) {
            e.preventDefault();
            const target = document.querySelector(this.getAttribute('href'));
            if (target) {
                target.scrollIntoView({
                    behavior: 'smooth',
                    block: 'start'
                });
            }
        });
    });
    
    // Navbar background on scroll
    const navbar = document.querySelector('.navbar');
    window.addEventListener('scroll', function() {
        if (window.scrollY > 100) {
            navbar.style.background = 'rgba(255, 255, 255, 0.98)';
        } else {
            navbar.style.background = 'rgba(255, 255, 255, 0.95)';
        }
    });
    
    // Terminal animation
    const terminal = document.querySelector('.terminal-window');
    if (terminal) {
        // Add typing animation to terminal
        const terminalLines = document.querySelectorAll('.terminal-line');
        terminalLines.forEach((line, index) => {
            line.style.opacity = '0';
            line.style.transform = 'translateY(10px)';
            
            setTimeout(() => {
                line.style.transition = 'all 0.5s ease';
                line.style.opacity = '1';
                line.style.transform = 'translateY(0)';
            }, index * 300);
        });
    }
    
    // Intersection Observer for animations
    const observerOptions = {
        threshold: 0.1,
        rootMargin: '0px 0px -50px 0px'
    };
    
    const observer = new IntersectionObserver(function(entries) {
        entries.forEach(entry => {
            if (entry.isIntersecting) {
                entry.target.classList.add('animate-in');
            }
        });
    }, observerOptions);
    
    // Observe elements for animation
    document.querySelectorAll('.feature-card, .doc-card, .step').forEach(el => {
        observer.observe(el);
    });
    
    // Copy code functionality (old method for code blocks without buttons)
    document.querySelectorAll('.code-block:not(.step .code-block)').forEach(block => {
        block.addEventListener('click', function() {
            const code = this.querySelector('code').textContent;
            navigator.clipboard.writeText(code).then(() => {
                // Show temporary feedback
                const original = this.style.background;
                this.style.background = 'rgba(102, 126, 234, 0.2)';
                setTimeout(() => {
                    this.style.background = original;
                }, 300);
            });
        });
        block.title = 'Click to copy';
        block.style.cursor = 'pointer';
    });
});

// Add animation classes
const style = document.createElement('style');
style.textContent = `
    .animate-in {
        animation: slideInUp 0.6s ease forwards;
    }
    
    @keyframes slideInUp {
        from {
            opacity: 0;
            transform: translateY(30px);
        }
        to {
            opacity: 1;
            transform: translateY(0);
        }
    }
    
    .nav-links.active {
        display: flex;
        position: absolute;
        top: 100%;
        left: 0;
        right: 0;
        background: white;
        flex-direction: column;
        padding: 1rem;
        box-shadow: 0 10px 30px rgba(0, 0, 0, 0.1);
    }
    
    @media (max-width: 768px) {
        .nav-links {
            display: none;
        }
    }
`;
document.head.appendChild(style);

// GitHub stars counter (optional)
function fetchGitHubStars() {
    fetch('https://api.github.com/repos/abdullahainun/tenangdb')
        .then(response => response.json())
        .then(data => {
            const stars = data.stargazers_count;
            const starsElement = document.getElementById('github-stars');
            if (starsElement && stars) {
                starsElement.textContent = `⭐ ${stars} stars`;
            }
        })
        .catch(error => {
            console.log('Could not fetch GitHub stars:', error);
        });
}

// Call on load
fetchGitHubStars();

// Copy code function for step buttons (old)
function copyCode(button) {
    const codeBlock = button.closest('.code-block');
    const code = codeBlock.querySelector('code').textContent;
    
    navigator.clipboard.writeText(code).then(() => {
        // Show feedback
        const originalText = button.textContent;
        button.textContent = '✅';
        button.style.color = '#4ade80';
        
        setTimeout(() => {
            button.textContent = originalText;
            button.style.color = '';
        }, 1000);
    }).catch(err => {
        console.error('Failed to copy: ', err);
        // Fallback for older browsers
        const textArea = document.createElement('textarea');
        textArea.value = code;
        document.body.appendChild(textArea);
        textArea.select();
        try {
            document.execCommand('copy');
            const originalText = button.textContent;
            button.textContent = '✅';
            button.style.color = '#4ade80';
            setTimeout(() => {
                button.textContent = originalText;
                button.style.color = '';
            }, 1000);
        } catch (err) {
            console.error('Fallback copy failed: ', err);
        }
        document.body.removeChild(textArea);
    });
}

// Copy code function for new unified steps
function copyStepCode(button, code) {
    navigator.clipboard.writeText(code).then(() => {
        // Show feedback
        const originalText = button.textContent;
        button.textContent = '✅';
        button.style.color = '#4ade80';
        
        setTimeout(() => {
            button.textContent = originalText;
            button.style.color = '';
        }, 1000);
    }).catch(err => {
        console.error('Failed to copy: ', err);
        // Fallback for older browsers
        const textArea = document.createElement('textarea');
        textArea.value = code;
        document.body.appendChild(textArea);
        textArea.select();
        try {
            document.execCommand('copy');
            const originalText = button.textContent;
            button.textContent = '✅';
            button.style.color = '#4ade80';
            setTimeout(() => {
                button.textContent = originalText;
                button.style.color = '';
            }, 1000);
        } catch (err) {
            console.error('Fallback copy failed: ', err);
        }
        document.body.removeChild(textArea);
    });
}