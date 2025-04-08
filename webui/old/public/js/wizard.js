/**
 * Agent Form Wizard - Navigation and UI functionality
 */
document.addEventListener('DOMContentLoaded', function() {
    // Check if the wizard exists on the page
    const wizardSidebar = document.querySelector('.wizard-sidebar');
    if (!wizardSidebar) return;

    // Get all sections and nav items
    const navItems = document.querySelectorAll('.wizard-nav-item');
    const sections = document.querySelectorAll('.form-section');
    const prevButton = document.getElementById('prevSection');
    const nextButton = document.getElementById('nextSection');
    const currentStepLabelEl = document.getElementById('currentStepLabel');
    const progressDotsContainer = document.getElementById('progressDots');
    
    // Create progress dots
    const totalSteps = sections.length;
    
    // Create dots for each section
    if (progressDotsContainer) {
        for (let i = 0; i < totalSteps; i++) {
            const dot = document.createElement('div');
            dot.className = 'progress-dot';
            dot.setAttribute('data-index', i);
            dot.addEventListener('click', () => setActiveSection(i));
            progressDotsContainer.appendChild(dot);
        }
    }
    
    // Get all progress dots
    const progressDots = document.querySelectorAll('.progress-dot');
    
    // Track current active section
    let currentSectionIndex = 0;
    
    // Initialize 
    updateNavigation();
    
    // Add click events to nav items
    navItems.forEach((item, index) => {
        item.addEventListener('click', () => {
            setActiveSection(index);
        });
    });
    
    // Add click events to prev/next buttons
    if (prevButton) {
        prevButton.addEventListener('click', () => {
            if (currentSectionIndex > 0) {
                setActiveSection(currentSectionIndex - 1);
            }
        });
    }
    
    if (nextButton) {
        nextButton.addEventListener('click', () => {
            if (currentSectionIndex < sections.length - 1) {
                setActiveSection(currentSectionIndex + 1);
            }
        });
    }
    
    /**
     * Set the active section and update navigation
     */
    function setActiveSection(index) {
        // Remove active class from all sections and nav items
        sections.forEach(section => section.classList.remove('active'));
        navItems.forEach(item => item.classList.remove('active'));
        progressDots.forEach(dot => dot.classList.remove('active'));
        
        // Add active class to current section, nav item, and dot
        sections[index].classList.add('active');
        navItems[index].classList.add('active');
        if (progressDots[index]) {
            progressDots[index].classList.add('active');
        }
        
        // Update current section index
        currentSectionIndex = index;
        
        // Update navigation state
        updateNavigation();
        
        // Scroll to top of section
        sections[index].scrollIntoView({behavior: 'smooth', block: 'start'});
    }
    
    /**
     * Update navigation buttons and progress
     */
    function updateNavigation() {
        // Update section label
        if (currentStepLabelEl && navItems[currentSectionIndex]) {
            // Extract text content without the icon
            const navText = navItems[currentSectionIndex].textContent.trim();
            currentStepLabelEl.textContent = navText;
        }
        
        // Update prev/next buttons
        if (prevButton) {
            prevButton.disabled = currentSectionIndex === 0;
            prevButton.style.opacity = currentSectionIndex === 0 ? 0.5 : 1;
        }
        
        if (nextButton) {
            nextButton.disabled = currentSectionIndex === sections.length - 1;
            nextButton.style.opacity = currentSectionIndex === sections.length - 1 ? 0.5 : 1;
            
            // Change text for last step
            if (currentSectionIndex === sections.length - 2) {
                nextButton.innerHTML = 'Finish <i class="fas fa-check"></i>';
            } else {
                nextButton.innerHTML = 'Next <i class="fas fa-arrow-right"></i>';
            }
        }
    }
    
    // Helper function to validate current section before proceeding
    function validateCurrentSection() {
        // Implement validation logic here based on the current section
        // Return true if valid, false if not
        return true;
    }
    
    // Add to initAgentFormCommon function if it exists
    if (typeof window.initAgentFormCommon === 'function') {
        const originalInit = window.initAgentFormCommon;
        
        window.initAgentFormCommon = function(options) {
            // Call the original initialization function
            originalInit(options);
            
            // Now initialize the wizard navigation
            setActiveSection(0);
        };
    }
});