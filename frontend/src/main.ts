import './lib/styles/tokens.css';
import './lib/styles/components.css';
import './app.css';
import App from './App.svelte';
import {mount} from 'svelte';

const target = document.getElementById('app');
const app = mount(App, {target: target as HTMLElement});

export default app;
