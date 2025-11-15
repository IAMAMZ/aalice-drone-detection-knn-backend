import React from "react";
import { Link, useLocation } from "react-router-dom";
import "./styles/Navigation.css";

function Navigation() {
  const location = useLocation();

  return (
    <nav className="Navigation">
      <Link
        to="/detection"
        className={`NavLink ${location.pathname === "/detection" ? "active" : ""}`}
      >
        Detection
      </Link>
      <Link
        to="/upload"
        className={`NavLink ${location.pathname === "/upload" ? "active" : ""}`}
      >
        Upload
      </Link>
      <Link
        to="/map"
        className={`NavLink ${location.pathname === "/map" ? "active" : ""}`}
      >
        Map
      </Link>
      <Link
        to="/science"
        className={`NavLink ${location.pathname === "/science" ? "active" : ""}`}
      >
        How It Works
      </Link>
    </nav>
  );
}

export default Navigation;

