/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

const SHOW_DELAY_MS = 150;
const MIN_VISIBLE_MS = 250;
const COMPLETE_ANIMATION_MS = 180;
const TRICKLE_INTERVAL_MS = 160;
const INITIAL_PROGRESS = 0.08;
const MAX_PROGRESS = 0.92;

const listeners = new Set();

let state = {
  visible: false,
  progress: 0,
  pending: 0,
};

let visibleSince = 0;
let showTimer = null;
let hideTimer = null;
let completeTimer = null;
let trickleTimer = null;

const emit = () => {
  listeners.forEach((listener) => listener());
};

const setState = (patch) => {
  state = { ...state, ...patch };
  emit();
};

const clearTimer = (timerRef) => {
  if (timerRef !== null) {
    window.clearTimeout(timerRef);
  }
  return null;
};

const clearIntervalTimer = (timerRef) => {
  if (timerRef !== null) {
    window.clearInterval(timerRef);
  }
  return null;
};

const stopTrickling = () => {
  trickleTimer = clearIntervalTimer(trickleTimer);
};

const startTrickling = () => {
  if (trickleTimer !== null) return;
  trickleTimer = window.setInterval(() => {
    if (!state.visible || state.pending <= 0) return;
    const remaining = Math.max(MAX_PROGRESS - state.progress, 0);
    if (remaining <= 0.001) return;
    const increment = Math.max(remaining * 0.18, 0.015);
    setState({
      progress: Math.min(state.progress + increment, MAX_PROGRESS),
    });
  }, TRICKLE_INTERVAL_MS);
};

const showLoading = () => {
  if (state.pending <= 0) return;
  showTimer = clearTimer(showTimer);
  hideTimer = clearTimer(hideTimer);
  completeTimer = clearTimer(completeTimer);
  visibleSince = Date.now();
  setState({
    visible: true,
    progress: Math.max(state.progress, INITIAL_PROGRESS),
  });
  startTrickling();
};

const scheduleShow = () => {
  if (state.visible || showTimer !== null) return;
  showTimer = window.setTimeout(() => {
    showTimer = null;
    showLoading();
  }, SHOW_DELAY_MS);
};

const finishLoading = () => {
  showTimer = clearTimer(showTimer);

  if (!state.visible) {
    stopTrickling();
    setState({ progress: 0 });
    return;
  }

  const elapsed = Date.now() - visibleSince;
  const remainingVisible = Math.max(MIN_VISIBLE_MS - elapsed, 0);

  hideTimer = clearTimer(hideTimer);
  hideTimer = window.setTimeout(() => {
    hideTimer = null;
    stopTrickling();
    setState({ progress: 1 });

    completeTimer = clearTimer(completeTimer);
    completeTimer = window.setTimeout(() => {
      completeTimer = null;
      if (state.pending > 0) {
        visibleSince = Date.now();
        setState({
          visible: true,
          progress: INITIAL_PROGRESS,
        });
        startTrickling();
        return;
      }
      setState({
        visible: false,
        progress: 0,
      });
    }, COMPLETE_ANIMATION_MS);
  }, remainingVisible);
};

export const subscribeGlobalLoading = (listener) => {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
};

export const getGlobalLoadingSnapshot = () => state;

export const beginGlobalLoading = () => {
  hideTimer = clearTimer(hideTimer);
  completeTimer = clearTimer(completeTimer);

  const nextPending = state.pending + 1;
  state = { ...state, pending: nextPending };

  if (state.visible) {
    if (state.progress >= 0.99) {
      setState({
        pending: nextPending,
        progress: INITIAL_PROGRESS,
      });
    } else {
      emit();
    }
    startTrickling();
    return;
  }

  emit();
  scheduleShow();
};

export const endGlobalLoading = () => {
  if (state.pending <= 0) return;

  const nextPending = state.pending - 1;
  state = { ...state, pending: nextPending };
  emit();

  if (nextPending > 0) {
    return;
  }

  finishLoading();
};
