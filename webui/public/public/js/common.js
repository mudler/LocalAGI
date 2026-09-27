 // Function to show toast notifications with enhanced animation
 function showToast(message, type) {
    const toast = document.getElementById('toast');
    const toastMessage = document.getElementById('toast-message');
    
    // Set message
    toastMessage.textContent = message;
    
    // Set toast type (success/error)
    toast.className = 'toast';
    toast.classList.add(type === 'success' ? 'toast-success' : 'toast-error');
    
    // Show toast with enhanced animation
    setTimeout(() => {
        toast.classList.add('toast-visible');
    }, 100);
    
    // Hide toast after 3 seconds with animation
    setTimeout(() => {
        toast.classList.remove('toast-visible');
        
        // Clean up after animation completes
        setTimeout(() => {
            toast.className = 'toast';
        }, 400);
    }, 3000);
}

// Function to create the glitch effect on headings
document.addEventListener('DOMContentLoaded', function() {
    const headings = document.querySelectorAll('h1');
    
    headings.forEach(heading => {
        heading.addEventListener('mouseover', function() {
            this.style.animation = 'glitch 0.3s infinite';
        });
        
        heading.addEventListener('mouseout', function() {
            this.style.animation = 'neonPulse 2s infinite';
        });
    });
    

});