"use client";

import { useEffect, useState } from "react";
import styles from "./styles/Listen.module.css";

const ListenButton = ({ isListening, onClick, disable }) => {
  return (
    <button
      disabled={disable}
      onClick={onClick}
      className={
        disable
          ? styles.ListenButton
          : isListening
          ? `${styles.ListenButton} ${styles.Enabled} ${styles.listening}`
          : `${styles.ListenButton} ${styles.Enabled}`
      }
    >
      <img 
        src={`${process.env.PUBLIC_URL}/Logo.png`}
        alt="AALIS Logo" 
        className={styles.logoIcon}
      />
    </button>
  );
};

const Listen = ({ disable, startListening, stopListening, isListening }) => {
  const [listen, setListen] = useState(false);

  useEffect(() => {
    if (isListening === false && listen === true) {
      setListen(false);
    }
  }, [isListening]);

  useEffect(() => {
    if (listen) {
      startListening();
    } else {
      if (isListening) {
        stopListening();
      }
    }
  }, [listen]);

  const toggleListen = () => {
    setListen(!listen);
  };

  return (
    <>
      <section className={styles.RippleContainer}>
        <div
          className={
            isListening
              ? `${styles.RippleBox} ${styles.RippleBoxPlay}`
              : `${styles.RippleBox} ${styles.RippleBoxStop}`
          }
        >
          <div
            className={
              isListening
                ? `${styles.RippleButton} ${styles.RippleButtonPlay}`
                : `${styles.RippleButton} ${styles.RippleButtonStop}`
            }
          >
            <ListenButton
              isListening={isListening}
              onClick={toggleListen}
              disable={disable}
            />
          </div>
        </div>
      </section>
      {/* <div
        className={
          isListening
            ? `${styles.CirlceItems} ${styles.Play}`
            : `${styles.CirlceItems} ${styles.Pause}`
        }
      >
        <div className={styles.CircleItem}></div>
        <div className={styles.CircleItem}></div>
        <div className={styles.CircleItem}></div>
        <ListenButton
          isListening={isListening}
          onClick={toggleListen}
          disable={disable}
        />
      </div> */}
    </>
  );
};

export default Listen;
