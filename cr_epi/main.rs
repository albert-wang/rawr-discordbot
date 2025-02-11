use crunchyroll_rs::{Crunchyroll, MediaCollection, Series};

use std::env;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Log in to Crunchyroll with your email and password.
    // Support for username login was dropped by Crunchyroll on December 6th, 2023
    let crunchyroll = Crunchyroll::builder().login_anonymously().await?;

    // Ducking christ, fine.
    // Get the series, season and episode from arguments
    let args: Vec<String> = env::args().collect();
    if args.len() != 4 {
        panic!("usage: <series> <season> <episode>");
    }

    let series_id: String = args[1].clone();
    let season: u32 = args[2].parse()?;
    let episode: String = args[3].clone();

    let series: Series = crunchyroll.media_from_id(&series_id).await?;
    let seasons = series.seasons().await?;
    let target_season = seasons.iter().find(|s| s.season_number == season).unwrap();

    let episodes = target_season.episodes().await?;
    let target_episode = episodes.iter().find(|e| e.episode == episode).unwrap();

    println!(
        "https://crunchyroll.com/watch/{}/{}",
        target_episode.id, target_episode.slug_title
    );
    Ok(())
}
