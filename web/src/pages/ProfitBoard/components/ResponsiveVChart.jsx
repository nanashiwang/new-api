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
import React, { useEffect, useMemo, useRef, useState } from 'react';
import { VChart } from '@visactor/react-vchart';

const ResponsiveVChart = ({
  chartKey,
  spec,
  onClick,
  option,
  minHeight = 320,
}) => {
  const containerRef = useRef(null);
  const chartRef = useRef(null);
  const sizeRef = useRef({ width: 0, height: minHeight });
  const [size, setSize] = useState({ width: 0, height: minHeight });

  useEffect(() => {
    sizeRef.current = size;
  }, [size]);

  useEffect(() => {
    const element = containerRef.current;
    if (!element) return undefined;

    const measure = () => {
      const rect = element.getBoundingClientRect();
      const nextWidth = Math.round(rect.width);
      const nextHeight = Math.max(Math.round(rect.height || 0), minHeight);
      setSize((prev) =>
        prev.width === nextWidth && prev.height === nextHeight
          ? prev
          : { width: nextWidth, height: nextHeight },
      );
    };

    measure();
    const rafId = window.requestAnimationFrame(measure);
    const timerId = window.setTimeout(measure, 80);
    const resizeObserver = new ResizeObserver(() => {
      window.requestAnimationFrame(measure);
    });
    resizeObserver.observe(element);

    let intersectionObserver;
    if ('IntersectionObserver' in window) {
      intersectionObserver = new IntersectionObserver((entries) => {
        if (entries[0]?.isIntersecting) {
          window.requestAnimationFrame(() => {
            measure();
            chartRef.current?.resize?.(
              sizeRef.current.width || undefined,
              sizeRef.current.height,
            );
          });
        }
      });
      intersectionObserver.observe(element);
    }

    return () => {
      window.cancelAnimationFrame(rafId);
      window.clearTimeout(timerId);
      resizeObserver.disconnect();
      intersectionObserver?.disconnect();
    };
  }, [minHeight]);

  useEffect(() => {
    if (!chartRef.current || !size.width) return undefined;
    const resizeChart = () => {
      chartRef.current?.resize?.(size.width, size.height);
    };
    const rafId = window.requestAnimationFrame(resizeChart);
    return () => window.cancelAnimationFrame(rafId);
  }, [chartKey, size.height, size.width]);

  const chartSpec = useMemo(
    () => ({
      ...spec,
      width: size.width || undefined,
      height: size.height,
    }),
    [size.height, size.width, spec],
  );

  return (
    <div
      ref={containerRef}
      className='w-full'
      style={{ minHeight }}
    >
      {size.width > 0 ? (
        <VChart
          key={chartKey}
          spec={chartSpec}
          option={option}
          onClick={onClick}
          onReady={(instance) => {
            chartRef.current = instance;
            window.requestAnimationFrame(() => {
              instance?.resize?.(size.width, size.height);
            });
          }}
        />
      ) : null}
    </div>
  );
};

export default ResponsiveVChart;
