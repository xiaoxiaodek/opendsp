import React from 'react';
import ReactDOM from 'react-dom/client';
import App from './App';

const rootEl = document.getElementById('root');
if (!rootEl) {
  document.body.innerHTML = '<h1>ERROR: #root not found</h1>';
} else {
  rootEl.innerHTML = `<div style="display:flex;justify-content:center;align-items:center;height:100vh;background:#f0f2f5;font-family:sans-serif">
    <div style="text-align:center">
      <div style="width:40px;height:40px;border:4px solid #e8e8e8;border-top-color:#1677ff;border-radius:50%;animation:spin .8s linear infinite;margin:0 auto 16px"></div>
      <div style="color:#999;font-size:14px">Loading...</div>
    </div>
    <style>@keyframes spin{to{transform:rotate(360deg)}}</style>
  </div>`;
  try {
    const root = ReactDOM.createRoot(rootEl);
    root.render(
      import.meta.env.DEV ? (
        <React.StrictMode>
          <App />
        </React.StrictMode>
      ) : <App />
    );
  } catch (e) {
    rootEl.innerHTML = '<h1 style="color:red;padding:20px">React Error: ' + String(e) + '</h1>';
  }
}
