import { useState, useEffect } from "react";
import { LogInfo, WindowHide, WindowSetSize } from "../wailsjs/runtime/runtime";
import { createHashRouter, RouterProvider } from "react-router-dom";
import Layout from "./components/Layout";
import SettingsPage from "./components/Settings/SettingsPage";
import { IsStartupComplete } from "../wailsjs/go/backend/App";

import { Loader2 } from "lucide-react";
import SearchContainer from "./components/Search/SearchContainer";


const LoadingSpinner = () => (
  <div className="flex justify-center items-center min-h-screen">
    <Loader2 className="h-8 w-8 animate-spin" />
  </div>
);


async function waitTillStartupComplete(resolve, reject) {
  const isStartupComplete = await IsStartupComplete();
  if (isStartupComplete) {
    resolve(true);
  } else {
    setTimeout(() => {
      waitTillStartupComplete(resolve, reject);
    }, 1000);
  }
}
async function waitTillReady() {
  return new Promise(async (resolve, reject) => {
    waitTillStartupComplete(resolve, reject);
  });
}


const router = createHashRouter([
  {
    path: "/",
    element: <Layout />,
    children: [
      {
        index: true,
        element: <SearchContainer />,
      },
      {
        path: "settings",
        element: <SettingsPage />,
      },
    ],
  },
]);

function App() {
  const [isBackendReady, setIsBackendReady] = useState(false);
  
  const [isStartupComplete, setIsStartupComplete] = useState(false);

  useEffect(() => {
    // Initialize both backend and Firebase
    const initialize = async () => {
      // Check if window.go is defined
      const checkBackend = () => {
        if (window.go) {
          setIsBackendReady(true);
        } else {
          // If not defined, check again after a short delay
          setTimeout(checkBackend, 100);
        }
      };

      // Initialize Firebase
      

      checkBackend();

      waitTillReady().then((isStartupComplete) => {
        setIsStartupComplete(isStartupComplete);
      });
    };

    initialize();
  }, []);

  // Show loading spinner while waiting for backend
  if (!isBackendReady || !isStartupComplete) {
    return <LoadingSpinner />;
  }

  return (
    <RouterProvider router={router} />
  );
}

export default App;
