document.addEventListener('DOMContentLoaded', () => {
    const playButton = document.getElementById('playButton');
    const saveButton = document.getElementById('saveButton');
    const outputLog = document.getElementById('outputLog');

    // Initially disable the buttons until the Go backend is ready
    playButton.disabled = true;
    saveButton.disabled = true;
    outputLog.textContent = 'Initializing Go backend...';

    const checkGoBackend = setInterval(() => {
        if (typeof window.RunEQCommand !== 'undefined' && typeof window.RunPlayCommand !== 'undefined') {
            clearInterval(checkGoBackend); // Stop checking once functions are available
            playButton.disabled = false; // Enable the buttons
            saveButton.disabled = false;
            outputLog.textContent = 'Go backend ready. Enter parameters and click Play or Save.';
            console.log('Go backend functions are ready.');

            playButton.addEventListener('click', async () => {
                const inputFile = document.getElementById('inputFile').value;
                const freqStart = parseFloat(document.getElementById('freqStart').value);
                const freqEnd = parseFloat(document.getElementById('freqEnd').value);
                const gain = parseFloat(document.getElementById('gain').value);

                outputLog.textContent = 'Starting real-time playback...';
                console.log('Calling Go RunPlayCommand with:', { inputFile, freqStart, freqEnd, gain });

                try {
                    const error = await window.RunPlayCommand(inputFile, freqStart, freqEnd, gain);
                    if (error) {
                        outputLog.textContent = `Error during playback:\n${error.message}`;
                        console.error('Go function returned an error:', error);
                    } else {
                        outputLog.textContent = 'Playback started. Check console for status.';
                        console.log('Go play function called successfully.');
                    }
                } catch (e) {
                    outputLog.textContent = `An unexpected error occurred:\n${e.message || e}`;
                    console.error('Unexpected error during Go function call:', e);
                }
            });

            saveButton.addEventListener('click', async () => {
                const inputFile = document.getElementById('inputFile').value;
                const outputFile = document.getElementById('outputFile').value;
                const freqStart = parseFloat(document.getElementById('freqStart').value);
                const freqEnd = parseFloat(document.getElementById('freqEnd').value);
                const gain = parseFloat(document.getElementById('gain').value);

                outputLog.textContent = 'Running EQ command and saving... Please wait.';
                console.log('Calling Go RunEQCommand with:', { inputFile, outputFile, freqStart, freqEnd, gain });

                try {
                    const error = await window.RunEQCommand(inputFile, outputFile, freqStart, freqEnd, gain);

                    if (error) {
                        outputLog.textContent = `Error saving file:\n${error.message}`;
                        console.error('Go function returned an error:', error);
                    } else {
                        outputLog.textContent = `EQ command finished successfully. Output saved to ${outputFile}`;
                        console.log('Go save function completed successfully.');
                    }
                } catch (e) {
                    outputLog.textContent = `An unexpected error occurred:\n${e.message || e}`;
                    console.error('Unexpected error during Go function call:', e);
                }
            });
        } else {
            console.log('Waiting for Go backend functions to be defined...');
        }
    }, 100);
});
