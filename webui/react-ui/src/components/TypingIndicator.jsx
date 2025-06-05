const TypingIndicator = () => {
  return (
    <span style={{ display: "inline-block", fontSize: "1rem" }}>
      <span className="dot">.</span>
      <span className="dot">.</span>
      <span className="dot">.</span>
      <style>
        {`
          .dot {
            animation: blink 1.5s infinite;
            font-weight: bold;
            opacity: 0.3;
          }

          .dot:nth-child(1) { animation-delay: 0s; }
          .dot:nth-child(2) { animation-delay: 0.3s; }
          .dot:nth-child(3) { animation-delay: 0.6s; }

          @keyframes blink {
            0% { opacity: 0.2; }
            50% { opacity: 1; }
            100% { opacity: 0.2; }
          }
        `}
      </style>
    </span>
  );
};

export default TypingIndicator;
