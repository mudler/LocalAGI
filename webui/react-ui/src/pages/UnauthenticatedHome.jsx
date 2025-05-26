import { useLogin, usePrivy } from "@privy-io/react-auth";
import React, { useEffect } from "react";

const UnauthenticatedHome = () => {
  const { ready, authenticated } = usePrivy();
  const { login } = useLogin({
    onComplete: () => {
      setTimeout(() => {
        window.location.reload();
      }, 1000);
    },
  });

  const disableLogin = !ready || (ready && authenticated);

  useEffect(() => {
    if (!disableLogin) {
      login();
    }
  }, [disableLogin]);

  return <div className="unauth-container"></div>;
};

export default UnauthenticatedHome;
