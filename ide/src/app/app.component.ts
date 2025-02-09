import {TuiIcon, TuiRoot, TuiTextfield} from "@taiga-ui/core";
import {NgForOf} from '@angular/common';
import {ChangeDetectionStrategy, Component} from '@angular/core';
import {FormsModule} from '@angular/forms';
import {TuiTabs} from '@taiga-ui/kit';
import {CodeEditorComponent, CodeModel} from '@ngstack/code-editor';
import {DEFAULT_EDITOR_OPTIONS} from './types';

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [
    TuiRoot,
    FormsModule,
    NgForOf,
    TuiIcon,
    TuiTabs,
    TuiTextfield,
    CodeEditorComponent,
  ],
  templateUrl: './app.component.html',
  styleUrl: './app.component.less',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AppComponent {
  theme = 'vs-dark';

  model: CodeModel = {
    language: 'json',
    uri: 'main.json',
    value: '{}'
  };

  options = DEFAULT_EDITOR_OPTIONS

  onCodeChanged(value: string) {
    console.log('CODE', value);
  }

  protected open = false;
  protected activeItemIndex = 0;

  protected items = Array.from({length: 5}, (_, i) => `Item #${i}`);

  protected add(): void {
    this.items = this.items.concat(`Item #${Date.now()}`);
  }

  protected remove(removed: string): void {
    const index = this.items.indexOf(removed);

    this.items = this.items.filter((item) => item !== removed);

    if (index <= this.activeItemIndex) {
      this.activeItemIndex = Math.max(this.activeItemIndex - 1, 0);
    }
  }
}
