
## lounch dummy server
sudo docker build -t py-server .
sudo docker run -p 5000 py-server

## lounch likec4 diogram
sudo docker run --rm -it -v $(pwd):/data -p 4321:4321 likec4/likec4 serve --listen 0.0.0.0