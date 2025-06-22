import React, { useEffect } from "react";

const ExportSection = ({ id }) => {
  useEffect(() => {
    console.log("ExportSection rendered with id:", id);
  }, [id]);

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
          href={`/settings/export/${id}`}
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
