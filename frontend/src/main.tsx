import React from 'react';
import ReactDOM from 'react-dom/client';
import { ConfigProvider } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import App from './app/App';
import './styles/tokens.css';
import './styles.css';

ReactDOM.createRoot(document.getElementById('root') as HTMLElement).render(
  <React.StrictMode>
    <ConfigProvider
      locale={zhCN}
      theme={{
        token: {
          colorPrimary: '#2563eb',
          colorInfo: '#0891b2',
          colorSuccess: '#16a34a',
          colorError: '#dc2626',
          colorWarning: '#d97706',
          colorText: '#0f172a',
          colorTextSecondary: '#64748b',
          colorBorder: '#cbd5e1',
          borderRadius: 6,
          controlHeight: 40,
          fontFamily: '"Fira Sans", "Inter", "Segoe UI", Arial, sans-serif',
          fontFamilyCode: '"Fira Code", Consolas, "SFMono-Regular", "Liberation Mono", monospace',
        },
        components: {
          Button: {
            controlHeight: 40,
            paddingInline: 16,
            fontWeight: 600,
          },
          DatePicker: {
            controlHeight: 40,
          },
          Input: {
            controlHeight: 40,
          },
          Select: {
            controlHeight: 40,
          },
        },
      }}
    >
      <App />
    </ConfigProvider>
  </React.StrictMode>,
);
