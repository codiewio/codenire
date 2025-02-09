import {NG_EVENT_PLUGINS} from "@taiga-ui/event-plugins";
import {provideAnimations} from "@angular/platform-browser/animations";
import {ApplicationConfig, provideZoneChangeDetection} from '@angular/core';
import {provideRouter, Routes} from '@angular/router';
import {provideAnimationsAsync} from '@angular/platform-browser/animations/async';
import {provideCodeEditor} from '@ngstack/code-editor';

const routes: Routes = []

export const appConfig: ApplicationConfig = {
  providers: [
    provideAnimations(),
    provideZoneChangeDetection({eventCoalescing: true}),
    provideRouter(routes),
    NG_EVENT_PLUGINS,

    provideAnimationsAsync(),

    // Configure Code Editor
    provideCodeEditor({
      // editorVersion: '0.46.0',
      // use local Monaco installation
      baseUrl: 'assets/monaco',
      // use local Typings Worker
      typingsWorkerUrl: 'assets/workers/typings-worker.js'
    })
  ]
};
