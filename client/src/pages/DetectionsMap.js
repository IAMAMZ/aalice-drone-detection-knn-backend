import React, { useEffect, useRef, useState } from "react";
import mapboxgl from "mapbox-gl";
import "mapbox-gl/dist/mapbox-gl.css";
import { createDroneMarker3D } from "../components/DroneMarker3D";
import styles from "./styles/DetectionsMap.module.css";
import { toast } from "react-toastify";

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

function DetectionsMap() {
  const mapContainer = useRef(null);
  const map = useRef(null);
  const model3DRef = useRef(null);
  const [mapLoaded, setMapLoaded] = useState(false);
  const [error, setError] = useState(null);
  const [detections, setDetections] = useState([]);
  const [topDetection, setTopDetection] = useState(null);
  const [selectedDetection, setSelectedDetection] = useState(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const mapboxToken = process.env.REACT_APP_MAPBOX_TOKEN || process.env.REACT_APP_MAP_BOX_TOKEN;
    
    if (!mapboxToken) {
      setError("Mapbox token not found. Please set REACT_APP_MAPBOX_TOKEN or REACT_APP_MAP_BOX_TOKEN in your .env file");
      return;
    }

    if (map.current) return;
    if (!mapContainer.current) return;

    mapboxgl.accessToken = mapboxToken;

    try {
      map.current = new mapboxgl.Map({
        container: mapContainer.current,
        style: "mapbox://styles/mapbox/satellite-streets-v12",
        center: [0, 0],
        zoom: 2,
        pitch: 60,
        bearing: -15,
        antialias: true,
        fadeDuration: 300,
        maxPitch: 85,
        minPitch: 0,
      });

      const nav = new mapboxgl.NavigationControl({
        visualizePitch: true,
        showZoom: true,
        showCompass: true,
      });
      map.current.addControl(nav, 'top-right');
      map.current.addControl(new mapboxgl.FullscreenControl(), 'top-right');

      map.current.on("style.load", () => {
        if (map.current.getSource('mapbox-dem')) {
          map.current.removeSource('mapbox-dem');
        }
        map.current.addSource('mapbox-dem', {
          'type': 'raster-dem',
          'url': 'mapbox://mapbox.mapbox-terrain-dem-v1',
          'tileSize': 512,
          'maxzoom': 14
        });

        map.current.setTerrain({ 
          'source': 'mapbox-dem', 
          'exaggeration': 3.0 
        });

        if (!map.current.getLayer('sky')) {
          map.current.addLayer({
            'id': 'sky',
            'type': 'sky',
            'paint': {
              'sky-type': 'atmosphere',
              'sky-atmosphere-sun': [0.0, 0.0],
              'sky-atmosphere-sun-intensity': 15,
              'sky-atmosphere-color': '#87CEEB'
            }
          });
        }

        map.current.setFog({
          'range': [0.5, 10],
          'color': '#1a1a2e',
          'horizon-blend': 0.1,
          'high-color': '#87CEEB',
          'space-color': '#1a1a2e',
          'star-intensity': 0.3
        });
      });

      map.current.on("load", () => {
        setMapLoaded(true);
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
      if (map.current) {
        map.current.remove();
        map.current = null;
      }
    };
  }, []);

  useEffect(() => {
    fetchDetections();
  }, []);

  useEffect(() => {
    if (map.current && mapLoaded && detections.length > 0) {
      renderAllDetectionsOnMap();
    }
  }, [mapLoaded, detections, selectedDetection]);

  useEffect(() => {
    if (selectedDetection && model3DRef.current) {
      render3DModel();
    }
  }, [selectedDetection]);

  const fetchDetections = async () => {
    try {
      setLoading(true);
      // Normalize URL to avoid double slashes
      const normalizedUrl = server.replace(/\/$/, '');
      const response = await fetch(`${normalizedUrl}/api/detections`);
      if (!response.ok) {
        throw new Error("Failed to fetch detections");
      }
      const data = await response.json();
      const detectionsWithLocation = data.filter(d => d.latitude != null && d.longitude != null);
      setDetections(detectionsWithLocation);
      
      if (detectionsWithLocation.length > 0) {
        const top = detectionsWithLocation.reduce((prev, current) => {
          return (current.confidence > prev.confidence) ? current : prev;
        });
        setTopDetection(top);
        setSelectedDetection(top);
        
        if (map.current) {
          // Fit bounds to show all detections
          const bounds = new mapboxgl.LngLatBounds();
          detectionsWithLocation.forEach(detection => {
            bounds.extend([detection.longitude, detection.latitude]);
          });
          
          map.current.fitBounds(bounds, {
            padding: 100,
            duration: 2000,
            maxZoom: 15
          });
        }
      } else {
        setTopDetection(null);
        setSelectedDetection(null);
      }
    } catch (error) {
      console.error("Error fetching detections:", error);
      setError(`Failed to load detections: ${error.message}`);
    } finally {
      setLoading(false);
    }
  };

  async function sendDroneSMS() {
    try {
      if (!detections || detections.length === 0) {
        toast.error("No prediction data available to send SMS");
        return;
      }

      const smsBody = JSON.stringify(detections);

      console.log("Sending SMS alert:", smsBody);

      const response = await fetch(`${server}/api/sms/send`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ body: smsBody }),
      });

      if (!response.ok) {
        const error = await response.json().catch(() => ({}));
        throw new Error(error.message || `HTTP error! status: ${response.status}`);
      }

      const result = await response.json();
      console.log("SMS sent successfully:", result);
      toast.success("SMS alert sent successfully!");

    } catch (error) {
      console.error("Error sending SMS:", error);
      toast.error(error.message || "Failed to send SMS alert");
    }
  }

  const getModelPath = (detection) => {
    // First, try to get model from metadata if available
    if (detection.metadata && detection.metadata.model) {
      return `/models/${detection.metadata.model}.glb`;
    }
    
    // Try to extract from predictions if available
    if (detection.predictions) {
      try {
        const predictions = typeof detection.predictions === 'string' 
          ? JSON.parse(detection.predictions) 
          : detection.predictions;
        
        if (Array.isArray(predictions) && predictions.length > 0) {
          const topPrediction = predictions[0];
          if (topPrediction.metadata && topPrediction.metadata.model) {
            return `/models/${topPrediction.metadata.model}.glb`;
          }
        }
      } catch (e) {
        console.warn('Failed to parse predictions:', e);
      }
    }
    
    // Map primaryLabel to model name
    const labelToModel = {
      'shahed-136': 'Shahed-136',
      'shahed': 'Shahed-136',
      'normal drone no noise shaheed': 'Shahed-136',
      'mq-9': 'MQ-9',
      'reaper': 'MQ-9',
      'rq-170': 'RQ-170',
      'sentinel': 'RQ-170',
      'fpv': 'FPV_Kamikaze',
      'kamikaze': 'FPV_Kamikaze',
      'fpv kamikaze': 'FPV_Kamikaze',
      'orion': 'BA-Orion',
      'ba-orion': 'BA-Orion',
      'hawk': 'USNT-Hawk',
      'usnt-hawk': 'USNT-Hawk',
      'uas': 'normalized_UAS',
      'normalized uas': 'normalized_UAS'
    };
    
    if (detection.primaryLabel) {
      const labelLower = detection.primaryLabel.toLowerCase();
      // Check for exact match first
      if (labelToModel[labelLower]) {
        return `/models/${labelToModel[labelLower]}.glb`;
      }
      // Check for partial match
      for (const [key, model] of Object.entries(labelToModel)) {
        if (labelLower.includes(key)) {
          return `/models/${model}.glb`;
        }
      }
    }
    
    // Default fallback
    return '/models/drone.glb';
  };

  const render3DModel = () => {
    if (!model3DRef.current || !selectedDetection) return;

    const modelPath = getModelPath(selectedDetection);
    
    // Clear previous content
    model3DRef.current.innerHTML = '';

    // Show loading state
    const loadingDiv = document.createElement('div');
    loadingDiv.style.cssText = 'display: flex; align-items: center; justify-content: center; height: 100%; color: #888; font-family: "Courier New", monospace;';
    loadingDiv.textContent = 'Loading 3D Model...';
    model3DRef.current.appendChild(loadingDiv);

    // Create 3D model viewer
    try {
      const viewer = createDroneMarker3D(modelPath, true);
      viewer.style.width = '100%';
      viewer.style.height = '100%';
      
      // Replace loading with viewer after a short delay to ensure it's ready
      setTimeout(() => {
        model3DRef.current.innerHTML = '';
        model3DRef.current.appendChild(viewer);
      }, 100);
    } catch (error) {
      console.error('Error creating 3D model viewer:', error);
      loadingDiv.textContent = '3D Model Unavailable';
      loadingDiv.style.color = '#ff0000';
    }
  };

  const renderAllDetectionsOnMap = () => {
    if (!map.current || detections.length === 0) return;

    // Remove existing layers and sources
    if (map.current.getLayer('detection-clusters')) {
      map.current.removeLayer('detection-clusters');
    }
    if (map.current.getLayer('detection-cluster-count')) {
      map.current.removeLayer('detection-cluster-count');
    }
    if (map.current.getLayer('detection-unclustered-point')) {
      map.current.removeLayer('detection-unclustered-point');
    }
    if (map.current.getLayer('detection-unclustered-point-selected')) {
      map.current.removeLayer('detection-unclustered-point-selected');
    }
    if (map.current.getSource('detections')) {
      map.current.removeSource('detections');
    }

    // Create GeoJSON from all detections
    const detectionsGeoJSON = {
      type: 'FeatureCollection',
      features: detections.map(detection => ({
        type: 'Feature',
        geometry: {
          type: 'Point',
          coordinates: [detection.longitude, detection.latitude]
        },
        properties: {
          id: detection.id,
          confidence: detection.confidence || 0,
          primaryType: detection.primaryType || 'TARGET',
          primaryLabel: detection.primaryLabel || 'DRONE',
          primaryCategory: detection.primaryCategory || 'Unknown',
          countryOfOrigin: detection.countryOfOrigin || '',
          snrDb: detection.snrDb || 0,
          timestamp: detection.timestamp
        }
      }))
    };

    map.current.addSource('detections', {
      type: 'geojson',
      data: detectionsGeoJSON,
      cluster: true,
      clusterMaxZoom: 14,
      clusterRadius: 50,
      clusterProperties: {
        maxConfidence: ['max', ['get', 'confidence']]
      }
    });

    // Add cluster circles
    map.current.addLayer({
      id: 'detection-clusters',
      type: 'circle',
      source: 'detections',
      filter: ['has', 'point_count'],
      paint: {
        'circle-color': [
          'step',
          ['get', 'point_count'],
          'rgba(127, 29, 29, 0.3)',
          10,
          'rgba(200, 50, 50, 0.4)',
          30,
          'rgba(255, 0, 0, 0.5)'
        ],
        'circle-radius': [
          'step',
          ['get', 'point_count'],
          20,
          10,
          30,
          30,
          40
        ],
        'circle-stroke-width': 2,
        'circle-stroke-color': 'rgba(255, 0, 0, 0.8)'
      }
    });

    // Add cluster count labels
    map.current.addLayer({
      id: 'detection-cluster-count',
      type: 'symbol',
      source: 'detections',
      filter: ['has', 'point_count'],
      layout: {
        'text-field': '{point_count_abbreviated}',
        'text-font': ['DIN Offc Pro Medium', 'Arial Unicode MS Bold'],
        'text-size': 12
      },
      paint: {
        'text-color': '#ffffff'
      }
    });

    // Add unclustered points
    map.current.addLayer({
      id: 'detection-unclustered-point',
      type: 'circle',
      source: 'detections',
      filter: ['!', ['has', 'point_count']],
      paint: {
        'circle-color': [
          'interpolate',
          ['linear'],
          ['get', 'confidence'],
          0, 'rgba(100, 100, 100, 0.5)',
          0.5, 'rgba(200, 100, 50, 0.6)',
          0.7, 'rgba(255, 50, 50, 0.7)',
          0.9, 'rgba(255, 0, 0, 0.8)'
        ],
        'circle-radius': [
          'interpolate',
          ['linear'],
          ['zoom'],
          10, 8,
          15, 12,
          20, 16
        ],
        'circle-stroke-width': 2,
        'circle-stroke-color': [
          'interpolate',
          ['linear'],
          ['get', 'confidence'],
          0, 'rgba(100, 100, 100, 0.8)',
          0.5, 'rgba(200, 100, 50, 0.9)',
          0.7, 'rgba(255, 50, 50, 1)',
          0.9, 'rgba(255, 0, 0, 1)'
        ]
      }
    });

    // Add selected point highlight
    if (selectedDetection) {
      const selectedGeoJSON = {
        type: 'FeatureCollection',
        features: [{
          type: 'Feature',
          geometry: {
            type: 'Point',
            coordinates: [selectedDetection.longitude, selectedDetection.latitude]
          },
          properties: {
            id: selectedDetection.id
          }
        }]
      };

      if (map.current.getSource('selected-detection')) {
        map.current.removeSource('selected-detection');
      }

      map.current.addSource('selected-detection', {
        type: 'geojson',
        data: selectedGeoJSON
      });

      map.current.addLayer({
        id: 'detection-unclustered-point-selected',
        type: 'circle',
        source: 'selected-detection',
        paint: {
          'circle-radius': [
            'interpolate',
            ['linear'],
            ['zoom'],
            10, 15,
            15, 25,
            20, 35
          ],
          'circle-color': 'rgba(255, 255, 0, 0.3)',
          'circle-stroke-width': 3,
          'circle-stroke-color': 'rgba(255, 255, 0, 1)'
        }
      });
    }

    // Add click handlers
    map.current.on('click', 'detection-clusters', (e) => {
      const features = map.current.queryRenderedFeatures(e.point, {
        layers: ['detection-clusters']
      });
      const clusterId = features[0].properties.cluster_id;
      const source = map.current.getSource('detections');
      source.getClusterExpansionZoom(clusterId, (err, zoom) => {
        if (err) return;
        map.current.easeTo({
          center: features[0].geometry.coordinates,
          zoom: zoom,
          duration: 500
        });
      });
    });

    map.current.on('click', 'detection-unclustered-point', (e) => {
      const feature = e.features[0];
      const detectionId = feature.properties.id;
      const detection = detections.find(d => d.id === detectionId);
      if (detection) {
        setSelectedDetection(detection);
        map.current.flyTo({
          center: [detection.longitude, detection.latitude],
          zoom: Math.max(map.current.getZoom(), 15),
          pitch: 70,
          bearing: -20,
          duration: 1000
        });
      }
    });

    // Change cursor on hover
    map.current.on('mouseenter', 'detection-clusters', () => {
      map.current.getCanvas().style.cursor = 'pointer';
    });
    map.current.on('mouseleave', 'detection-clusters', () => {
      map.current.getCanvas().style.cursor = '';
    });
    map.current.on('mouseenter', 'detection-unclustered-point', () => {
      map.current.getCanvas().style.cursor = 'pointer';
    });
    map.current.on('mouseleave', 'detection-unclustered-point', () => {
      map.current.getCanvas().style.cursor = '';
    });
  };

  if (error) {
    return (
      <div className={styles.mapContainer}>
        <div className={styles.placeholder}>
          <p>{error}</p>
        </div>
      </div>
    );
  }

  return (
    <div className={styles.pageContainer}>
      <div className={styles.header}>
        <h2>Drone Detection Map</h2>
        <div className={styles.stats}>
          <span>{detections.length} Total Detections</span>
          {topDetection && (
            <>
              <span className={styles.topConfidence}>
                Top: {(topDetection.confidence * 100).toFixed(1)}%
              </span>
              <span className={styles.coordinates}>
                {topDetection.model || 'Unknown Model'}
              </span>
            </>
          )}
          <button onClick={fetchDetections} className={styles.refreshButton}>
            Refresh
          </button>
          <button onClick={sendDroneSMS}>
            Send SMS
          </button>
        </div>
      </div>
      <div className={styles.mapContainer}>
        {loading && (
          <div className={styles.loadingOverlay}>
            <p>Loading detections...</p>
          </div>
        )}
        <div ref={mapContainer} className={styles.map} />
        {detections.length === 0 && !loading && (
          <div className={styles.placeholder}>
            <p>No detections found</p>
            <small>Detections will appear here once drones are detected with location data</small>
          </div>
        )}
        {selectedDetection && !loading && (
          <div className={styles.detectionInfoBox}>
            <div className={styles.detectionInfoHeader}>
              <h3>{selectedDetection.primaryType || 'TARGET'}</h3>
              <button onClick={() => setSelectedDetection(null)} className={styles.closeButton}>×</button>
            </div>
            <div className={styles.model3DContainer} ref={model3DRef}>
              {/* 3D Model will be rendered here */}
            </div>
            <div className={styles.detectionInfoContent}>
              <div className={styles.modelInfo}>
                <div className={styles.modelName}>
                  {(() => {
                    const modelPath = getModelPath(selectedDetection);
                    const modelName = modelPath.replace('/models/', '').replace('.glb', '');
                    return modelName !== 'drone' ? modelName : (selectedDetection.primaryLabel || 'Unknown Model');
                  })()}
                </div>
                <div className={styles.modelLabel}>{selectedDetection.primaryLabel || selectedDetection.primaryType || 'DRONE'}</div>
              </div>
              <div className={styles.confidenceBar}>
                <div className={styles.confidenceLabel}>
                  <span>CONFIDENCE</span>
                  <span className={styles.confidenceValue}>{(selectedDetection.confidence * 100).toFixed(0)}%</span>
                </div>
                <div className={styles.progressBar}>
                  <div 
                    className={styles.progressFill} 
                    style={{ width: `${selectedDetection.confidence * 100}%` }}
                  />
                </div>
              </div>
              <div className={styles.infoGrid}>
                <div className={styles.infoItem}>
                  <span className={styles.infoLabel}>CATEGORY</span>
                  <span className={styles.infoValue}>{selectedDetection.primaryCategory || 'Unknown'}</span>
                </div>
                {selectedDetection.countryOfOrigin && (
                  <div className={styles.infoItem}>
                    <span className={styles.infoLabel}>ORIGIN</span>
                    <span className={styles.infoValue}>{selectedDetection.countryOfOrigin}</span>
                  </div>
                )}
                <div className={styles.infoItem}>
                  <span className={styles.infoLabel}>LATITUDE</span>
                  <span className={styles.infoValue}>{selectedDetection.latitude.toFixed(6)}</span>
                </div>
                <div className={styles.infoItem}>
                  <span className={styles.infoLabel}>LONGITUDE</span>
                  <span className={styles.infoValue}>{selectedDetection.longitude.toFixed(6)}</span>
                </div>
                {selectedDetection.snrDb && (
                  <div className={styles.infoItem}>
                    <span className={styles.infoLabel}>SNR</span>
                    <span className={styles.infoValue}>{selectedDetection.snrDb.toFixed(2)} dB</span>
                  </div>
                )}
                <div className={styles.infoItem}>
                  <span className={styles.infoLabel}>TIMESTAMP</span>
                  <span className={styles.infoValue}>{new Date(selectedDetection.timestamp).toLocaleTimeString()}</span>
                </div>
              </div>
              {detections.length > 1 && (
                <div className={styles.navigationButtons}>
                  <button 
                    onClick={() => {
                      const currentIndex = detections.findIndex(d => d.id === selectedDetection.id);
                      const prevIndex = currentIndex > 0 ? currentIndex - 1 : detections.length - 1;
                      const prevDetection = detections[prevIndex];
                      setSelectedDetection(prevDetection);
                      if (map.current) {
                        map.current.flyTo({
                          center: [prevDetection.longitude, prevDetection.latitude],
                          zoom: Math.max(map.current.getZoom(), 15),
                          pitch: 70,
                          bearing: -20,
                          duration: 1000
                        });
                      }
                    }}
                    className={styles.navButton}
                  >
                    ← Previous
                  </button>
                  <span className={styles.detectionCounter}>
                    {detections.findIndex(d => d.id === selectedDetection.id) + 1} / {detections.length}
                  </span>
                  <button 
                    onClick={() => {
                      const currentIndex = detections.findIndex(d => d.id === selectedDetection.id);
                      const nextIndex = currentIndex < detections.length - 1 ? currentIndex + 1 : 0;
                      const nextDetection = detections[nextIndex];
                      setSelectedDetection(nextDetection);
                      if (map.current) {
                        map.current.flyTo({
                          center: [nextDetection.longitude, nextDetection.latitude],
                          zoom: Math.max(map.current.getZoom(), 15),
                          pitch: 70,
                          bearing: -20,
                          duration: 1000
                        });
                      }
                    }}
                    className={styles.navButton}
                  >
                    Next →
                  </button>
                </div>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

export default DetectionsMap;