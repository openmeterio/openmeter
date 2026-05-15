import React from 'react'
import ReactDOM from 'react-dom/client'
import { BrowserRouter, Routes, Route } from 'react-router-dom'
import './index.css'
import 'highlight.js/styles/atom-one-dark.min.css'
import HomePage from './pages/HomePage'
import LocalPage from './pages/LocalPage'
import CoverPage from './pages/CoverPage'
import ReportPage from './pages/ReportPage'
import NotFoundPage from './pages/NotFoundPage'

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<HomePage />} />
        <Route path="/local" element={<LocalPage />} />
        <Route path="/r/:token" element={<CoverPage />} />
        <Route path="/r/:token/details" element={<ReportPage />} />
        <Route path="*" element={<NotFoundPage />} />
      </Routes>
    </BrowserRouter>
  </React.StrictMode>,
)
