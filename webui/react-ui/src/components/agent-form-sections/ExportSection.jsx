import React, { useEffect } from "react";

const ExportSection = ({ agentName }) => {
  useEffect(() => {
    console.log("ExportSection rendered with agentName:", agentName);
  }, [agentName]);

  return (
    <div className="">
      <div className="section-title">
        <h2>Export Data</h2>
      </div>

      <div className="section-content">
        <p className="section-description">
          Export your agent configuration for backup or transfer.
        </p>
        <a
          href={`/settings/export/${agentName}`}
          className="action-btn"
          style={{
            display: "inline-flex",
            alignItems: "center",
            textDecoration: "none",
          }}
        >
          <i className="fas fa-file-export"></i> Export Configuration
        </a>
      </div>
    </div>
  );
};

export default ExportSection;
