import React from 'react';

/**
 * SendMail action component
 */
const SendMailAction = ({ index, onActionConfigChange, getConfigValue }) => {
  return (
    <div className="send-mail-action">
      <div className="form-group mb-3">
        <label htmlFor={`email${index}`}>Email</label>
        <input
          type="email"
          id={`email${index}`}
          value={getConfigValue('email', '')}
          onChange={(e) => onActionConfigChange('email', e.target.value)}
          className="form-control"
          placeholder="your-email@example.com"
        />
        <small className="form-text text-muted">Email address to send from</small>
      </div>
      
      <div className="form-group mb-3">
        <label htmlFor={`username${index}`}>Username</label>
        <input
          type="text"
          id={`username${index}`}
          value={getConfigValue('username', '')}
          onChange={(e) => onActionConfigChange('username', e.target.value)}
          className="form-control"
          placeholder="SMTP username (often same as email)"
        />
      </div>
      
      <div className="form-group mb-3">
        <label htmlFor={`password${index}`}>Password</label>
        <input
          type="password"
          id={`password${index}`}
          value={getConfigValue('password', '')}
          onChange={(e) => onActionConfigChange('password', e.target.value)}
          className="form-control"
          placeholder="SMTP password or app password"
        />
        <small className="form-text text-muted">For Gmail, use an app password</small>
      </div>
      
      <div className="form-group mb-3">
        <label htmlFor={`smtpHost${index}`}>SMTP Host</label>
        <input
          type="text"
          id={`smtpHost${index}`}
          value={getConfigValue('smtpHost', '')}
          onChange={(e) => onActionConfigChange('smtpHost', e.target.value)}
          className="form-control"
          placeholder="smtp.gmail.com"
        />
      </div>
      
      <div className="form-group mb-3">
        <label htmlFor={`smtpPort${index}`}>SMTP Port</label>
        <input
          type="text"
          id={`smtpPort${index}`}
          value={getConfigValue('smtpPort', '587')}
          onChange={(e) => onActionConfigChange('smtpPort', e.target.value)}
          className="form-control"
          placeholder="587"
        />
        <small className="form-text text-muted">Common ports: 587 (TLS), 465 (SSL)</small>
      </div>
    </div>
  );
};

export default SendMailAction;
