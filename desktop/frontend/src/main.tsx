import React from 'react'
import {createRoot} from 'react-dom/client'
import './style.css'
import LocalizedApp from './LocalizedApp'
import ErrorBoundary from './ErrorBoundary'

const container = document.getElementById('root')

const root = createRoot(container!)

root.render(
    <React.StrictMode>
        <ErrorBoundary>
            <LocalizedApp/>
        </ErrorBoundary>
    </React.StrictMode>
)
