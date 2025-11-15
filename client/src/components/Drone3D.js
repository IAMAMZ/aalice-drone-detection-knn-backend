import React, { Suspense, useRef, useState, useCallback, useEffect } from 'react';
import { Canvas } from '@react-three/fiber';
import { OrbitControls, useGLTF } from '@react-three/drei';
import styles from './styles/Drone3D.module.css';

function DroneModel({ modelPath, onLoad }) {
  const { scene } = useGLTF(modelPath);
  const droneRef = useRef();

  useEffect(() => {
    if (scene && onLoad) {
      onLoad();
    }
  }, [scene, onLoad]);

  return (
    <primitive 
      ref={droneRef}
      object={scene} 
      scale={1} 
      position={[0, 0, 0]}
      rotation={[0, 0, 0]}
    />
  );
}

function Drone3D({ modelPath = '/models/drone.glb', showControls = true }) {
  const [isLoading, setIsLoading] = useState(true);
  const [hasError, setHasError] = useState(false);
  
  const handleLoad = useCallback(() => {
    setIsLoading(false);
    setHasError(false);
  }, []);

  useEffect(() => {
    const timeout = setTimeout(() => {
      if (isLoading) {
        console.warn('Model loading timeout, hiding loading overlay');
        setIsLoading(false);
      }
    }, 5000);

    return () => clearTimeout(timeout);
  }, [isLoading]);

  return (
    <div className={styles.container}>
      {isLoading && !hasError && (
        <div className={styles.loading}>
          Loading 3D Model...
        </div>
      )}
      {hasError && (
        <div className={styles.error}>
          <div>Failed to load 3D model</div>
          <div className={styles.errorMessage}>
            Make sure drone.glb is in /public/models/
          </div>
        </div>
      )}
      <Canvas
        camera={{ position: [5, 5, 5], fov: 50 }}
        gl={{ antialias: true }}
        style={{ width: '100%', height: '100%', display: 'block' }}
        onError={(error) => {
          console.error('Canvas error:', error);
          setHasError(true);
          setIsLoading(false);
        }}
      >
        <ambientLight intensity={0.5} />
        <directionalLight position={[10, 10, 5]} intensity={1} />
        <pointLight position={[-10, -10, -5]} intensity={0.5} />
        
        <Suspense 
          fallback={null}
        >
          <DroneModel modelPath={modelPath} onLoad={handleLoad} />
          {showControls && <OrbitControls enableDamping dampingFactor={0.05} />}
        </Suspense>
        
        <gridHelper args={[10, 10]} />
      </Canvas>
    </div>
  );
}

useGLTF.preload('/models/drone.glb');

export default Drone3D;

