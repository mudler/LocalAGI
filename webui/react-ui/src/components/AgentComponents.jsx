/**
 * Status indicator component for displaying agent status
 * @param {string} status - Status text to display (e.g., "Active", "Paused")
 * @param {string} color - Color for the status indicator (e.g., "#22c55e" for green)
 */
export const AgentStatus = ({ status, color }) => {
  return (
    <span
      className="status-indicator"
      style={{ color }}
    >
      <span
        className="status-dot"
        style={{ background: color }}
      ></span>
      {status}
    </span>
  );
};

/**
 * Action buttons component for agent control actions
 * @param {object} agent - The agent object containing properties like 'active'
 * @param {boolean} loading - Whether the action is in progress
 * @param {function} onPauseResume - Handler for pause/resume button click
 * @param {function} onDelete - Handler for delete button click
 */
export const AgentActionButtons = ({
                                     agent,
                                     loading,
                                     onPauseResume,
                                     onDelete
                                   }) => {
  return (
    <div className="action-buttons">
      <button
        className="action-btn pause-resume-btn"
        onClick={onPauseResume}
        disabled={loading}
      >
        <i
          className={`fas ${agent?.active ? "fa-pause" : "fa-play"}`}
        ></i>{" "}
        {agent?.active ? "Pause Agent" : "Resume Agent"}
      </button>
      <button
        className="action-btn delete-btn"
        onClick={onDelete}
        disabled={loading}
      >
        <i className="fas fa-trash"></i> Delete Agent
      </button>
    </div>
  );
};

/**
 * Single action button component for reusable standalone buttons
 * @param {string} text - Button text to display
 * @param {string} icon - FontAwesome icon class (e.g., "fa-plus")
 * @param {function} onClick - Handler for button click
 * @param {boolean} loading - Whether the action is in progress
 * @param {boolean} disabled - Whether the button should be disabled
 * @param {string} variant - Button style variant ("default", "pause-resume", "delete")
 */
export const ActionButton = ({
                               text,
                               icon,
                               onClick,
                               loading = false,
                               disabled = false,
                               variant = "default"
                             }) => {
  // Determine button class based on variant
  const buttonClass = variant === "delete"
    ? "action-btn delete-btn"
    : variant === "pause-resume"
      ? "action-btn pause-resume-btn"
      : "action-btn";

  return (
    <button
      className={buttonClass}
      onClick={onClick}
      disabled={loading || disabled}
    >
      {loading ? (
        <>
          <i className="fas fa-spinner fa-spin"></i>{" "}
          Loading...
        </>
      ) : (
        <>
          {icon && <i className={`fas ${icon}`}></i>}{" "}
          {text}
        </>
      )}
    </button>
  );
};

/**
 * Container component for action buttons
 * @param {React.ReactNode} children - Button components to render
 */
export const ActionButtonsContainer = ({ children }) => {
  return (
    <div className="action-buttons">
      {children}
    </div>
  );
};
