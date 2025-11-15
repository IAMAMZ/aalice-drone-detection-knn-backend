import React from "react";
import PrototypeUpload from "../components/PrototypeUpload";

const getBackendUrl = () => {
  if (process.env.REACT_APP_BACKEND_URL) {
    return process.env.REACT_APP_BACKEND_URL;
  }
  if (window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1') {
    return 'http://localhost:5001';
  }
  return window.location.origin;
};

const server = getBackendUrl();

function UploadPage({ onUploadSuccess }) {
  return (
    <div className="UploadPage">
      <div className="MainGrid">
        <section className="GridColumn">
          <PrototypeUpload backendUrl={server} onUploadSuccess={onUploadSuccess} />
        </section>
      </div>
    </div>
  );
}

export default UploadPage;

