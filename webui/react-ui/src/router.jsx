import { createBrowserRouter } from "react-router-dom";
import App from "./App";
import ConditionalHome from "./pages/ConditionalHome";
import AgentSettings from "./pages/AgentSettings";
import AgentsList from "./pages/AgentsList";
import CreateAgent from "./pages/CreateAgent";
import Chat from "./pages/Chat";
import ActionsPlayground from "./pages/ActionsPlayground";
import GroupCreate from "./pages/GroupCreate";
import AgentStatus from "./pages/AgentStatus";
import ImportAgent from "./pages/ImportAgent";

const BASE_URL = import.meta.env.BASE_URL || "/app";

export const router = createBrowserRouter(
  [
    {
      path: "/",
      element: <App />,
      children: [
        {
          index: true,
          element: <ConditionalHome />,
        },
        {
          path: "agents",
          element: <AgentsList />,
        },
        {
          path: "create",
          element: <CreateAgent />,
        },
        {
          path: "settings/:id",
          element: <AgentSettings />,
        },
        {
          path: "talk/:id",
          element: <Chat />,
        },
        {
          path: "actions-playground",
          element: <ActionsPlayground />,
        },
        {
          path: "group-create",
          element: <GroupCreate />,
        },
        {
          path: "import",
          element: <ImportAgent />,
        },
        {
          path: "status/:id",
          element: <AgentStatus />,
        },
      ],
    },
  ],
  {
    basename: BASE_URL,
  }
);
