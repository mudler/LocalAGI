import { usePrivy } from "@privy-io/react-auth";
import { Navigate } from "react-router-dom";

function ProtectedRoute({ children }) {
  const { ready, authenticated } = usePrivy();

  // Show loading while auth state is being determined
  if (!ready) {
    return (
      <div className="loading-container">
        <div className="spinner"></div>
      </div>
    );
  }

  // Redirect to home if not authenticated
  if (!authenticated) {
    return <Navigate to="/" replace />;
  }

  // Render the protected component if authenticated
  return children;
}

export default ProtectedRoute; 