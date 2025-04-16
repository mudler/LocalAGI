/**
 * Header component for page titles and descriptions
 *
 * @param {string} icon - FontAwesome icon class (e.g., "fa-comments")
 * @param {string} title - The main title text
 * @param {string} description - Descriptive text below the title
 * @param {string} name - Optional name to be highlighted (e.g., agent name)
 * @returns {JSX.Element} Header component
 */
const Header = ({
                  icon = 'fas fa-comments',
                  title = 'Chat with',
                  description = 'Send messages and interact with your agent in real time.',
                  name = ''
                }) => {
  return (
    <div className="header-content">
      <i className={`${icon} header-icon`} />
      <div>
        <div className="header-title">
          {title}{" "}
          {name && (
            <span className="header-title-highlight">
              {title === 'Agent Settings' ? `- ${name}` : name}
            </span>
          )}
        </div>
        <div className="header-description">
          {description}
        </div>
      </div>
    </div>
  );
};

export default Header;
