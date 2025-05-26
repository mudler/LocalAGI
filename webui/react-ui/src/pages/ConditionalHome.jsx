import React from "react";
import { usePrivy } from "@privy-io/react-auth";
import Home from "../pages/Home";
import UnauthenticatedHome from "../pages/UnauthenticatedHome";

const ConditionalHome = () => {
  const { ready, authenticated } = usePrivy();

  if (!ready) {
    return <div>Loading...</div>;
  }

  return authenticated ? <Home /> : <UnauthenticatedHome />;
};

export default ConditionalHome;
