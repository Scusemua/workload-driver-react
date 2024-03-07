import * as React from 'react';
import App from '@app/index';
import { render } from '@testing-library/react';
import '@testing-library/jest-dom';
import fetch from 'jest-fetch-mock';

describe('App tests', () => {
  beforeEach(() => {
    fetch.resetMocks();
  });

  test('should render default App component', () => {
    fetch.mockResponseOnce(
      JSON.stringify([
        {
          NodeId: 'distributed-notebook-worker',
          Pods: [
            {
              PodName: '62677bbf-359a-4f0b-96e7-6baf7ac65545-7ad16',
              PodPhase: 'running',
              PodAge: '127h2m45s',
              PodIP: '10.0.0.1',
            },
          ],
          Age: '147h4m53s',
          IP: '172.20.0.3',
          CapacityCPU: 64,
          CapacityMemory: 64000,
          CapacityGPUs: 8,
          CapacityVGPUs: 72,
          AllocatedCPU: 0.24,
          AllocatedMemory: 1557.1,
          AllocatedGPUs: 2,
          AllocatedVGPUs: 4,
        },
      ]),
    );

    const { asFragment } = render(<App />);

    expect(asFragment()).toBeDefined();
  });
});
