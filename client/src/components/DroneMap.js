import React, { useEffect, useRef, useState } from "react";
import mapboxgl from "mapbox-gl";
import "mapbox-gl/dist/mapbox-gl.css";
import styles from "./styles/DroneMap.module.css";

const DroneMap = ({ latitude, longitude, classification, modelInfo }) => {
  const mapContainer = useRef(null);
  const map = useRef(null);
  const marker = useRef(null);
  const infoPanel = useRef(null);
  const [mapLoaded, setMapLoaded] = useState(false);
  const [error, setError] = useState(null);
  const [showInfo, setShowInfo] = useState(false);

  useEffect(() => {
    const mapboxToken = process.env.REACT_APP_MAPBOX_TOKEN || process.env.REACT_APP_MAP_BOX_TOKEN;
    
    console.log("Mapbox token check:", {
      hasToken: !!mapboxToken,
      tokenLength: mapboxToken ? mapboxToken.length : 0,
      tokenPrefix: mapboxToken ? mapboxToken.substring(0, 10) : "none"
    });
    
    if (!mapboxToken) {
      setError("Mapbox token not found. Please set REACT_APP_MAPBOX_TOKEN or REACT_APP_MAP_BOX_TOKEN in your .env file");
      console.warn("Mapbox token not found. Please set REACT_APP_MAPBOX_TOKEN or REACT_APP_MAP_BOX_TOKEN in your .env file");
      return;
    }

    if (map.current) return;
    if (!mapContainer.current) {
      console.log("Map container not ready yet, will retry");
      return;
    }

    mapboxgl.accessToken = mapboxToken;

    const initialCenter = (longitude != null && latitude != null) 
      ? [longitude, latitude] 
      : [0, 0];
    const initialZoom = (longitude != null && latitude != null) ? 15 : 2;

    try {
      map.current = new mapboxgl.Map({
        container: mapContainer.current,
        style: "mapbox://styles/mapbox/satellite-streets-v12",
        center: initialCenter,
        zoom: initialZoom,
        pitch: 65,
        bearing: -15,
      });

      map.current.on("style.load", () => {
        // Add 3D terrain
        if (map.current.getSource('mapbox-dem')) {
          map.current.removeSource('mapbox-dem');
        }
        map.current.addSource('mapbox-dem', {
          'type': 'raster-dem',
          'url': 'mapbox://mapbox.mapbox-terrain-dem-v1',
          'tileSize': 512,
          'maxzoom': 14
        });
        map.current.setTerrain({ 'source': 'mapbox-dem', 'exaggeration': 2.5 });
      });

      map.current.on("load", () => {
        setMapLoaded(true);
        console.log("Map loaded successfully");
        setError(null);
      });

      map.current.on("error", (e) => {
        console.error("Map error:", e);
        const errorMsg = e.error?.message || "Failed to load map";
        setError(`Map error: ${errorMsg}. Please check your Mapbox token.`);
      });
    } catch (err) {
      console.error("Error initializing map:", err);
      setError(`Failed to initialize map: ${err.message || err}`);
    }

    return () => {
      if (marker.current) {
        marker.current.remove();
        marker.current = null;
      }
      if (map.current) {
        map.current.remove();
        map.current = null;
      }
      setShowInfo(false);
    };
  }, [latitude, longitude]);

  useEffect(() => {
    if (!map.current || !mapLoaded) return;
    if (latitude == null || longitude == null) return;

    // Remove existing marker
    if (marker.current) {
      marker.current.remove();
      marker.current = null;
    }

    // Show info panel when new location is detected
    setShowInfo(true);

    // Create 2D drone icon marker using drone.svg
    const markerElement = document.createElement('div');
    markerElement.className = styles.droneIcon;
    markerElement.style.cursor = 'pointer';
    markerElement.innerHTML = `
      <img src="/image/drone.svg" alt="Drone" style="width: 50px; height: 50px;" />
    `;

    marker.current = new mapboxgl.Marker({
      element: markerElement,
      anchor: "center",
    })
      .setLngLat([longitude, latitude])
      .addTo(map.current);

    // Add click event to marker to toggle info panel
    markerElement.addEventListener('click', () => {
      setShowInfo(prev => !prev);
    });

    // Fly to location with 3D perspective
    map.current.flyTo({
      center: [longitude, latitude],
      zoom: 17,
      pitch: 75,
      bearing: -20,
      duration: 2000,
    });

  }, [latitude, longitude, mapLoaded, classification, modelInfo]);

  // Get the top prediction (highest confidence)
  const topPrediction = classification?.predictions?.[0];
  const metadata = topPrediction?.metadata;
  const threatAssessment = topPrediction?.threatAssessment;

  if (error) {
    return (
      <div className={styles.mapContainer}>
        <div className={styles.placeholder}>
          <p>{error}</p>
          <small>Make sure REACT_APP_MAPBOX_TOKEN is set in your .env file and restart the dev server</small>
        </div>
      </div>
    );
  }

  return (
    <div className={styles.mapContainer} style={{ position: 'relative' }}>
      <div ref={mapContainer} className={styles.map} />
      {latitude == null || longitude == null ? (
        <div className={styles.placeholder} style={{ position: 'absolute', top: 0, left: 0, right: 0, bottom: 0, zIndex: 1 }}>
          <p>Waiting for location data...</p>
          <small>Location will appear here when audio is analyzed</small>
        </div>
      ) : null}
      
      {showInfo && latitude != null && longitude != null && topPrediction && (
        <div 
          ref={infoPanel}
          className={styles.infoPanel}
          style={{
            position: 'absolute',
            top: '10px',
            right: '10px',
            width: '300px',
            maxHeight: 'calc(100% - 20px)',
            overflowY: 'auto',
            backgroundColor: 'rgba(0, 0, 0, 0.95)',
            border: '2px solid #dc3545',
            borderRadius: '8px',
            padding: '16px',
            zIndex: 1000,
            fontFamily: "'Segoe UI', Tahoma, Geneva, Verdana, sans-serif",
            color: '#fff',
            boxShadow: '0 0 30px rgba(220, 53, 69, 0.4)',
            fontSize: '0.85em'
          }}
        >
          <button
            onClick={() => setShowInfo(false)}
            style={{
              position: 'absolute',
              top: '8px',
              right: '8px',
              background: 'transparent',
              border: 'none',
              color: '#dc3545',
              fontSize: '20px',
              cursor: 'pointer',
              padding: '0',
              width: '24px',
              height: '24px',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              lineHeight: '1',
              fontWeight: 'bold'
            }}
          >
            ×
          </button>

          <div style={{ marginBottom: '12px', paddingRight: '30px' }}>
            <strong style={{ 
              fontSize: '1.1em', 
              display: 'block', 
              marginBottom: '6px',
              color: classification?.isDrone ? '#dc3545' : '#ffc107'
            }}>
              {classification?.isDrone ? ' DRONE DETECTED' : ' TARGET ACQUIRED'}
            </strong>
          </div>
          
          <div style={{ marginBottom: '10px', padding: '10px', background: 'rgba(220, 53, 69, 0.15)', borderRadius: '6px', borderLeft: '3px solid #dc3545' }}>
            <strong style={{ display: 'block', marginBottom: '6px', fontSize: '0.9em', color: '#dc3545' }}>IDENTIFICATION</strong>
            <div style={{ fontSize: '0.85em', lineHeight: '1.6' }}>
              <div><strong>Model:</strong> {metadata?.model || topPrediction.label}</div>
              <div><strong>Type:</strong> {topPrediction.type || 'Unknown'}</div>
              <div><strong>Category:</strong> {topPrediction.category}</div>
              <div><strong>Confidence:</strong> {(topPrediction.confidence * 100).toFixed(1)}%</div>
            </div>
          </div>
          
          <div style={{ marginBottom: '10px', padding: '10px', background: 'rgba(255, 193, 7, 0.1)', borderRadius: '6px', borderLeft: '3px solid #ffc107' }}>
            <strong style={{ display: 'block', marginBottom: '6px', fontSize: '0.9em', color: '#ffc107' }}>LOCATION</strong>
            <div style={{ fontSize: '0.8em', lineHeight: '1.6', fontFamily: 'monospace' }}>
              <div><strong>LAT:</strong> {latitude.toFixed(6)}</div>
              <div><strong>LNG:</strong> {longitude.toFixed(6)}</div>
            </div>
          </div>

          {metadata && (
            <div style={{ marginBottom: '10px', padding: '10px', background: 'rgba(13, 110, 253, 0.1)', borderRadius: '6px', borderLeft: '3px solid #0d6efd' }}>
              <strong style={{ display: 'block', marginBottom: '6px', fontSize: '0.9em', color: '#0d6efd' }}>SPECIFICATIONS</strong>
              <div style={{ fontSize: '0.8em', lineHeight: '1.6' }}>
                {metadata.manufacturer && <div><strong>Manufacturer:</strong> {metadata.manufacturer}</div>}
                {metadata.rotor_count && <div><strong>Rotors:</strong> {metadata.rotor_count}</div>}
                {metadata.max_range_km && <div><strong>Max Range:</strong> {metadata.max_range_km} km</div>}
                {metadata.max_speed_ms && <div><strong>Max Speed:</strong> {metadata.max_speed_ms} m/s</div>}
                {metadata.flight_time_minutes && <div><strong>Flight Time:</strong> {metadata.flight_time_minutes} min</div>}
                {metadata.payload_capacity_kg && <div><strong>Payload:</strong> {metadata.payload_capacity_kg} kg</div>}
              </div>
            </div>
          )}

          {threatAssessment && (
            <div style={{ marginBottom: '10px', padding: '10px', background: 'rgba(220, 53, 69, 0.2)', borderRadius: '6px', borderLeft: '3px solid #dc3545' }}>
              <strong style={{ display: 'block', marginBottom: '6px', fontSize: '0.9em', color: '#dc3545' }}>THREAT ASSESSMENT</strong>
              <div style={{ fontSize: '0.8em', lineHeight: '1.6' }}>
                {threatAssessment.threatLevel && (
                  <div>
                    <strong>Threat Level:</strong>{' '}
                    <span style={{
                      color: threatAssessment.threatLevel === 'high' ? '#dc3545' : 
                             threatAssessment.threatLevel === 'medium' ? '#ffc107' : '#28a745',
                      fontWeight: 'bold',
                      textTransform: 'uppercase'
                    }}>
                      {threatAssessment.threatLevel}
                    </span>
                  </div>
                )}
                {threatAssessment.riskCategory && (
                  <div><strong>Risk Category:</strong> {threatAssessment.riskCategory.replace(/_/g, ' ')}</div>
                )}
                {threatAssessment.jammingSusceptible !== undefined && (
                  <div><strong>Jamming Susceptible:</strong> {threatAssessment.jammingSusceptible ? 'Yes' : 'No'}</div>
                )}
                {threatAssessment.countermeasureRecommendations && (
                  <div style={{ marginTop: '6px', paddingTop: '6px', borderTop: '1px solid rgba(255,255,255,0.1)' }}>
                    <strong>Countermeasures:</strong><br/>
                    <span style={{ fontSize: '0.95em', lineHeight: '1.4' }}>
                      {threatAssessment.countermeasureRecommendations}
                    </span>
                  </div>
                )}
              </div>
            </div>
          )}
          
          <div style={{ padding: '10px', background: 'rgba(108, 117, 125, 0.2)', borderRadius: '6px', fontSize: '0.75em', color: '#adb5bd' }}>
            <div><strong>Timestamp:</strong> {new Date().toLocaleString()}</div>
            <div><strong>Status:</strong> {classification?.isDrone ? 'ACTIVE THREAT' : 'MONITORING'}</div>
          </div>
        </div>
      )}
    </div>
  );
};

export default DroneMap;