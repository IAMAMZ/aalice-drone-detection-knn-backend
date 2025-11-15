import React, { useState } from "react";
import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import { ToastContainer, Slide } from "react-toastify";
import "react-toastify/dist/ReactToastify.css";
import DetectionPage from "./pages/DetectionPage";
import UploadPage from "./pages/UploadPage";
import DetectionsMap from "./pages/DetectionsMap";
import SciencePage from "./pages/SciencePage";
import Navigation from "./components/Navigation";
import ChatbotToggle from "./components/ChatbotToggle";

function App() {
  const [modelInfo, setModelInfo] = useState(null);

  function handleUploadSuccess(payload) {
    if (payload?.stats) {
      setModelInfo(payload.stats);
    }
  }


  return (
    <BrowserRouter>
      <div className="App">
        <header className="TopHeader">
          <div>
            <h2>AALIS</h2>
            <p className="subtitle">Acoustic Autonomous Lightweight Interception System</p>
            <p>Advanced acoustic detection and threat assessment for autonomous drone interception.</p>
          </div>
          {modelInfo && (
            <div className="HeaderStats">
              <span>{modelInfo.prototypeCount} signatures</span>
              <span>{modelInfo.labelCount} classes</span>
              {modelInfo.usingExample && (
                <small>Example prototype set loaded</small>
              )}
            </div>
          )}
        </header>

        <Navigation />

        <Routes>
          <Route path="/" element={<Navigate to="/detection" replace />} />
          <Route
            path="/detection"
            element={<DetectionPage modelInfo={modelInfo} setModelInfo={setModelInfo} />}
          />
        
          <Route
            path="/upload"
            element={<UploadPage onUploadSuccess={handleUploadSuccess} />}
          />
          <Route
            path="/map"
            element={<DetectionsMap />}
          />
          <Route
            path="/science"
            element={<SciencePage />}
          />
        </Routes>

        {/* Add chatbot toggle to all pages */}
        <ChatbotToggle />

        <ToastContainer
          position="top-center"
          autoClose={5000}
          hideProgressBar
          newestOnTop={false}
          closeOnClick
          rtl={false}
          pauseOnFocusLoss
          pauseOnHover
          theme="dark"
          transition={Slide}
        />
      </div>
    </BrowserRouter>
  );
}

export default App;