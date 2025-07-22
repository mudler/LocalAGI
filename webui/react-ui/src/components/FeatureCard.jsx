import { Link } from "react-router-dom";

function FeatureCard({ to, imageSrc, imageAlt, title, description, authenticated, onLogin }) {
  const cardContent = (
    <>
      <img src={imageSrc} alt={imageAlt} />
      <div className="feature-content">
        <h3>{title}</h3>
        <p>{description}</p>
      </div>
    </>
  );

  if (authenticated) {
    return (
      <Link to={to} className="feature-card">
        {cardContent}
      </Link>
    );
  }

  return (
    <div className="feature-card" onClick={onLogin} style={{ cursor: 'pointer' }}>
      {cardContent}
    </div>
  );
}

export default FeatureCard; 