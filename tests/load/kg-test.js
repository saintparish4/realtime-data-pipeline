import { check, sleep } from "k6";
import { WebSocket } from "k6/ws";
import http from "k6/http";

export let options = {
  stages: [
    { duration: "2m", target: 100 },
    { duration: "5m", target: 100 },
    { duration: "2m", target: 200 },
    { duration: "5m", target: 200 },
    { duration: "2m", target: 300 },
    { duration: "5m", target: 300 },
    { duration: "2m", target: 0 },
  ],
};

export default function () {
  // Test HTTP API
  let response = http.get(
    "http://localhost:8080/api/v1/crypto/BTCUSDT/current"
  );
  check(response, { "status was 200": (r) => r.status === 200 });

  // Test Websocket Connection
  const ws = new WebSocket("ws://localhost:8080/ws/BTCUSDT");

  ws.onopen = function () {
    console.log("WebSocket connection opened");
  };

  ws.onmessage = function (event) {
    const data = JSON.parse(event.data);
    check(data, { "has symbol": (d) => d.symbol === "BTCUSDT" });
  };

  ws.onclose = function () {
    console.log("WebSocket connection closed");
  };

  sleep(1);
}
